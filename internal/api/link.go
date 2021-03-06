package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/photoprism/photoprism/internal/config"
	"github.com/photoprism/photoprism/internal/entity"
	"github.com/photoprism/photoprism/internal/event"
	"github.com/photoprism/photoprism/internal/form"
	"github.com/photoprism/photoprism/internal/query"
	"github.com/photoprism/photoprism/pkg/txt"
)

// newLink returns a new link entity initialized with request data
func newLink(c *gin.Context) (link entity.Link, err error) {
	var f form.NewLink

	if err := c.BindJSON(&f); err != nil {
		return link, err
	}

	link = entity.NewLink(f.Password, f.CanComment, f.CanEdit)

	if f.Expires > 0 {
		expires := time.Now().Add(time.Duration(f.Expires) * time.Second)
		link.LinkExpires = &expires
	}

	return link, nil
}

// POST /api/v1/albums/:uuid/link
func LinkAlbum(router *gin.RouterGroup, conf *config.Config) {
	router.POST("/albums/:uuid/link", func(c *gin.Context) {
		if Unauthorized(c, conf) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrUnauthorized)
			return
		}

		m, err := query.AlbumByUUID(c.Param("uuid"))

		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, ErrAlbumNotFound)
			return
		}

		if link, err := newLink(c); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": txt.UcFirst(err.Error())})
			return
		} else {
			entity.Db().Model(&m).Association("Links").Append(link)
		}

		event.Success("created album share link")

		c.JSON(http.StatusOK, m)
	})
}

// POST /api/v1/photos/:uuid/link
func LinkPhoto(router *gin.RouterGroup, conf *config.Config) {
	router.POST("/photos/:uuid/link", func(c *gin.Context) {
		if Unauthorized(c, conf) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrUnauthorized)
			return
		}

		m, err := query.PhotoByUUID(c.Param("uuid"))

		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, ErrPhotoNotFound)
			return
		}

		if link, err := newLink(c); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": txt.UcFirst(err.Error())})
			return
		} else {
			entity.Db().Model(&m).Association("Links").Append(link)
		}

		event.Success("created photo share link")

		c.JSON(http.StatusOK, m)
	})
}

// POST /api/v1/labels/:uuid/link
func LinkLabel(router *gin.RouterGroup, conf *config.Config) {
	router.POST("/labels/:uuid/link", func(c *gin.Context) {
		if Unauthorized(c, conf) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrUnauthorized)
			return
		}

		m, err := query.LabelByUUID(c.Param("uuid"))

		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, ErrLabelNotFound)
			return
		}

		if link, err := newLink(c); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": txt.UcFirst(err.Error())})
			return
		} else {
			entity.Db().Model(&m).Association("Links").Append(link)
		}

		event.Success("created label share link")

		c.JSON(http.StatusOK, m)
	})
}
