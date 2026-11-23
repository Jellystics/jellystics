package handler

import (
	"errors"
	"net/http"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type itemRoute struct {
	Type     string  `json:"type"`
	Id       string  `json:"id"`
	SeriesId *string `json:"seriesId,omitempty"`
	AlbumId  *string `json:"albumId,omitempty"`
}

// ItemRoute resolves any media id to the kind of page that displays it:
// movie, series, album, artist, episode (with its series) or track (with its
// album). Anything else, including an episode without series or a track
// without album, is a 404.
//
// GET /api/v2/items/:id/route
func (h *V2Handler) ItemRoute(c *gin.Context) {
	id := c.Param("id")
	db := h.db.WithContext(c.Request.Context())
	found := func(err error) (bool, bool) { // (found, failed)
		if err == nil {
			return true, false
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false
		}
		v2Fail(c, err)
		return false, true
	}

	var item models.JFLibraryItem
	if ok, failed := found(db.Select("Id", "Type").Where(`"Id" = ?`, id).Take(&item).Error); failed {
		return
	} else if ok {
		kind := ""
		if item.Type != nil {
			switch *item.Type {
			case "Movie":
				kind = "movie"
			case "Series":
				kind = "series"
			case "MusicAlbum":
				kind = "album"
			case "MusicArtist":
				kind = "artist"
			}
		}
		if kind == "" {
			v2NotFound(c)
			return
		}
		c.JSON(http.StatusOK, itemRoute{Type: kind, Id: id})
		return
	}

	var ep models.JFLibraryEpisode
	if ok, failed := found(db.Select("Id", "SeriesId").Where(`"Id" = ? OR "EpisodeId" = ?`, id, id).Take(&ep).Error); failed {
		return
	} else if ok {
		if ep.SeriesId == nil || *ep.SeriesId == "" {
			v2NotFound(c)
			return
		}
		c.JSON(http.StatusOK, itemRoute{Type: "episode", Id: id, SeriesId: ep.SeriesId})
		return
	}

	var tr models.JFMusicTrack
	if ok, failed := found(db.Select("Id", "AlbumId").Where(`"Id" = ?`, id).Take(&tr).Error); failed {
		return
	} else if ok {
		if tr.AlbumId == nil || *tr.AlbumId == "" {
			v2NotFound(c)
			return
		}
		c.JSON(http.StatusOK, itemRoute{Type: "track", Id: id, AlbumId: tr.AlbumId})
		return
	}

	var ar models.JFMusicArtist
	if ok, failed := found(db.Select("Id").Where(`"Id" = ?`, id).Take(&ar).Error); failed {
		return
	} else if ok {
		c.JSON(http.StatusOK, itemRoute{Type: "artist", Id: id})
		return
	}
	v2NotFound(c)
}
