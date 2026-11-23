package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type searchItem struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"` // movie | series | album | artist | episode | track
	Name       string  `json:"name"`
	Year       *int    `json:"year,omitempty"`
	ParentID   *string `json:"parentId,omitempty"`
	ParentName *string `json:"parentName,omitempty"`
}

type searchUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const searchLimit = 8

// GET /api/v2/search?q= — catalog items and users whose name contains q
// (case-insensitive), at most 8 of each; prefix matches and top-level kinds
// (movie, series, artist, album) rank first.
func (h *V2Handler) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	out := gin.H{"items": []searchItem{}, "users": []searchUser{}}
	if q == "" {
		c.JSON(http.StatusOK, out)
		return
	}
	if len([]rune(q)) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q too long"})
		return
	}
	like := likePattern(q)
	prefix := strings.TrimSuffix(likePattern(q), "%")[1:] + "%"

	items := []searchItem{}
	if v2Fail(c, h.raw(c, &items, `
		SELECT id, kind, name, year, parent_id, parent_name FROM (
		  SELECT i."Id" AS id,
		         CASE i."Type" WHEN 'Movie' THEN 'movie' WHEN 'Series' THEN 'series' ELSE 'album' END AS kind,
		         i."Name" AS name, i."ProductionYear" AS year, i."ArtistId" AS parent_id,
		         CASE WHEN i."Type" = 'MusicAlbum' THEN i."AlbumArtist" END AS parent_name, 0 AS prio
		  FROM jf_library_items i
		  WHERE NOT i.archived AND i."Type" IN ('Movie', 'Series', 'MusicAlbum') AND i."Name" ILIKE ?
		  UNION ALL
		  SELECT ar."Id", 'artist', ar."Name", NULL, NULL, NULL, 0
		  FROM jf_music_artists ar WHERE NOT ar.archived AND ar."Name" ILIKE ?
		  UNION ALL
		  SELECT e."Id", 'episode', e."Name", e."ProductionYear", e."SeriesId", e."SeriesName", 1
		  FROM jf_library_episodes e WHERE NOT e.archived AND e."Name" ILIKE ?
		  UNION ALL
		  SELECT t."Id", 'track', t."Name", t."ProductionYear", t."AlbumId", t."AlbumName", 1
		  FROM jf_music_tracks t WHERE NOT t.archived AND t."Name" ILIKE ?
		) s
		ORDER BY (name ILIKE ?) DESC, prio, length(name), name, id
		LIMIT ?`, like, like, like, like, prefix, searchLimit)) {
		return
	}
	for i := range items {
		if items[i].Kind == "movie" || items[i].Kind == "series" {
			items[i].ParentID = nil
		}
	}
	users := []searchUser{}
	if v2Fail(c, h.raw(c, &users, `
		SELECT "Id" AS id, "Name" AS name FROM jf_users WHERE "Name" ILIKE ?
		ORDER BY ("Name" ILIKE ?) DESC, length("Name"), "Name" LIMIT ?`, like, prefix, searchLimit)) {
		return
	}
	out["items"], out["users"] = items, users
	c.JSON(http.StatusOK, out)
}
