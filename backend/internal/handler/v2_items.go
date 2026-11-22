package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ItemRouteHandler resolves any media id to the kind of page that displays it.
type ItemRouteHandler struct {
	db *gorm.DB
}

func NewItemRouteHandler(db *gorm.DB) *ItemRouteHandler { return &ItemRouteHandler{db: db} }

type itemRoute struct {
	Type     string  `json:"type"`
	Id       string  `json:"id"`
	SeriesId *string `json:"seriesId,omitempty"`
	AlbumId  *string `json:"albumId,omitempty"`
}

// GET /api/v2/items/:id/route
func (h *ItemRouteHandler) Route(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var item models.JFLibraryItem
	err := h.db.WithContext(ctx).Select("Id", "Type").Where(`"Id" = ?`, id).Take(&item).Error
	if err == nil {
		kind := "other"
		if item.Type != nil {
			switch *item.Type {
			case "Movie":
				kind = "movie"
			case "Series":
				kind = "series"
			case "MusicAlbum":
				kind = "album"
			}
		}
		c.JSON(http.StatusOK, itemRoute{Type: kind, Id: id})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		internalError(c, err)
		return
	}

	var ep models.JFLibraryEpisode
	err = h.db.WithContext(ctx).Select("Id", "SeriesId").Where(`"Id" = ? OR "EpisodeId" = ?`, id, id).Take(&ep).Error
	if err == nil {
		c.JSON(http.StatusOK, itemRoute{Type: "episode", Id: id, SeriesId: ep.SeriesId})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		internalError(c, err)
		return
	}

	var tr models.JFMusicTrack
	err = h.db.WithContext(ctx).Select("Id", "AlbumId").Where(`"Id" = ?`, id).Take(&tr).Error
	if err == nil {
		c.JSON(http.StatusOK, itemRoute{Type: "track", Id: id, AlbumId: tr.AlbumId})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	internalError(c, err)
}

func internalError(c *gin.Context, err error) {
	log.Printf("item route %s: %v", c.Param("id"), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
