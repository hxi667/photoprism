package classify

import (
	"bufio"
	"bytes"
	"errors"
	"image"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/photoprism/photoprism/pkg/txt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

// TensorFlow is a wrapper for tensorflow low-level API.
type TensorFlow struct {
	model      *tf.SavedModel
	modelsPath string
	disabled   bool
	modelName  string
	modelTags  []string
	labels     []string
}

// New returns new TensorFlow instance with Nasnet model.
func New(modelsPath string, disabled bool) *TensorFlow {
	return &TensorFlow{modelsPath: modelsPath, disabled: disabled, modelName: "nasnet", modelTags: []string{"photoprism"}}
}

// Init initialises tensorflow models if not disabled
func (t *TensorFlow) Init() (err error) {
	if t.disabled {
		return nil
	}

	return t.loadModel()
}

// File returns matching labels for a jpeg media file.
func (t *TensorFlow) File(filename string) (result Labels, err error) {
	if t.disabled {
		return result, nil
	}

	imageBuffer, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	return t.Labels(imageBuffer)
}

// Labels returns matching labels for a jpeg media string.
func (t *TensorFlow) Labels(img []byte) (result Labels, err error) {
	if t.disabled {
		return result, nil
	}

	if err := t.loadModel(); err != nil {
		return nil, err
	}

	// Make tensor
	tensor, err := t.makeTensor(img, "jpeg")

	if err != nil {
		log.Error(err)
		return nil, errors.New("invalid image")
	}

	// Run inference
	output, err := t.model.Session.Run(
		map[tf.Output]*tf.Tensor{
			t.model.Graph.Operation("input_1").Output(0): tensor,
		},
		[]tf.Output{
			t.model.Graph.Operation("predictions/Softmax").Output(0),
		},
		nil)

	if err != nil {
		log.Error(err)
		return result, errors.New("could not run inference")
	}

	if len(output) < 1 {
		return result, errors.New("result is empty")
	}

	// Return best labels
	result = t.bestLabels(output[0].Value().([][]float32)[0])

	if len(result) > 0 {
		log.Debugf("tensorflow: image classified as %+v", result)
	}

	return result, nil
}

func (t *TensorFlow) loadLabels(path string) error {
	modelLabels := path + "/labels.txt"

	log.Infof("tensorflow: loading classification labels from labels.txt")

	// Load labels
	f, err := os.Open(modelLabels)

	if err != nil {
		return err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Labels are separated by newlines
	for scanner.Scan() {
		t.labels = append(t.labels, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (t *TensorFlow) ModelLoaded() bool {
	return t.model != nil
}

func (t *TensorFlow) loadModel() error {
	if t.ModelLoaded() {
		return nil
	}

	modelPath := path.Join(t.modelsPath, t.modelName)

	log.Infof("tensorflow: loading image classification model from %s", txt.Quote(filepath.Base(modelPath)))

	// Load model
	model, err := tf.LoadSavedModel(modelPath, t.modelTags, nil)

	if err != nil {
		return err
	}

	t.model = model

	return t.loadLabels(modelPath)
}

// bestLabels returns the best 5 labels (if enough high probability labels) from the prediction of the model
func (t *TensorFlow) bestLabels(probabilities []float32) Labels {
	var result Labels

	for i, p := range probabilities {
		if i >= len(t.labels) {
			// break if probabilities and labels does not match
			break
		}

		// discard labels with low probabilities
		if p < 0.1 {
			continue
		}

		labelText := strings.ToLower(t.labels[i])

		rule := rules.Find(labelText)

		// discard labels that don't met the threshold
		if p < rule.Threshold {
			continue
		}

		// Get rule label name instead of t.labels name if it exists
		if rule.Label != "" {
			labelText = rule.Label
		}

		labelText = strings.TrimSpace(labelText)

		uncertainty := 100 - int(math.Round(float64(p*100)))

		result = append(result, Label{Name: labelText, Source: SrcImage, Uncertainty: uncertainty, Priority: rule.Priority, Categories: rule.Categories})
	}

	// Sort by probability
	sort.Sort(result)

	// return only the 5 best labels
	if l := len(result); l < 5 {
		return result[:l]
	} else {
		return result[:5]
	}
}

// makeTensor converts bytes jpeg image in a tensor object required as tensorflow model input
func (t *TensorFlow) makeTensor(image []byte, imageFormat string) (*tf.Tensor, error) {
	img, err := imaging.Decode(bytes.NewReader(image), imaging.AutoOrientation(true))

	if err != nil {
		return nil, err
	}

	width, height := 224, 224

	img = imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	return imageToTensorTF(img, width, height)
}

func imageToTensorTF(img image.Image, imageHeight, imageWidth int) (*tf.Tensor, error) {
	var tfImage [1][][][3]float32

	for j := 0; j < imageHeight; j++ {
		tfImage[0] = append(tfImage[0], make([][3]float32, imageWidth))
	}

	for i := 0; i < imageWidth; i++ {
		for j := 0; j < imageHeight; j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			tfImage[0][j][i][0] = convertTF(r)
			tfImage[0][j][i][1] = convertTF(g)
			tfImage[0][j][i][2] = convertTF(b)
		}
	}

	return tf.NewTensor(tfImage)
}

func convertTF(value uint32) float32 {
	return (float32(value>>8) - float32(127.5)) / float32(127.5)
}
