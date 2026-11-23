package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

// TestImportBackup_DurationIsSeconds verifies the PlaybackReporting TSV import
// stores PlayDuration verbatim as seconds. The plugin already reports seconds,
// so dividing by 10,000,000 (ticks) would wipe out the watch time.
func TestImportBackup_DurationIsSeconds(t *testing.T) {
	db := testutil.NewDB(t)

	// TSV: DateCreated \t UserId \t ItemId \t ItemType \t ItemName \t Method \t Client \t Device \t PlayDuration
	tsv := "DateCreated\tUserId\tItemId\tItemType\tItemName\tMethod\tClient\tDevice\tPlayDuration\n" +
		"2026-07-12\tuser-1\titem-1\tMovie\tInception\tDirectPlay\tWeb\tChrome\t600\n"

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("file", "backup.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(tsv)); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	gin.SetMode(gin.TestMode)
	h := handler.NewTasksApiHandler(nil, repository.New(db), nil)
	r := gin.New()
	r.POST("/import", h.ImportBackup)

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	var got models.JFPlaybackActivity
	if err := db.Where(`"Id" = ?`, "import-user-1-item-1-2026-07-12").First(&got).Error; err != nil {
		t.Fatalf("imported row not found: %v", err)
	}
	if got.PlaybackDuration == nil || *got.PlaybackDuration != 600 {
		t.Errorf("PlaybackDuration = %v, want 600 seconds (must not divide by ticks)", got.PlaybackDuration)
	}
	if got.Source != "import" {
		t.Errorf("Source = %q, want import", got.Source)
	}
	if !got.Imported {
		t.Error("Imported flag should be true")
	}
}

func postTSV(t *testing.T, r *gin.Engine, tsv string) map[string]any {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "backup.tsv")
	_, _ = fw.Write([]byte(tsv))
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// One play = one row: a TSV row already recorded by another source (plugin
// play, plugin PBR import...) is skipped and counted as such.
func TestImportBackup_SkipsPlaysCoveredByAnotherSource(t *testing.T) {
	t.Setenv("TZ", "UTC")
	const tsv = "DateCreated\tUserId\tItemId\tItemType\tItemName\tMethod\tClient\tDevice\tPlayDuration\n" +
		"2026-07-12 20:00:20\tuser-1\titem-1\tMovie\tInception\tDirectPlay\tWeb\tChrome\t600\n"
	start := time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC) // DateCreated - 20 s
	for _, source := range []string{"", "import-pbr", "jellystics-plugin"} {
		t.Run("existing="+source, func(t *testing.T) {
			db := testutil.NewDB(t)
			if source != "" {
				end := start.Add(10 * time.Minute)
				dur := int64(600)
				if err := db.Create(&models.JFPlaybackActivity{Id: "existing", UserId: sp("user-1"), ItemId: sp("item-1"), MediaType: sp("Movie"),
					PlaybackDuration: &dur, StartedAt: tp(start.Add(5 * time.Second)), EndedAt: &end, Source: source, Imported: source == "import-pbr"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.POST("/import", handler.NewTasksApiHandler(nil, repository.New(db), nil).ImportBackup)
			out := postTSV(t, r, tsv)
			var n int64
			db.Model(&models.JFPlaybackActivity{}).Count(&n)
			if n != 1 {
				t.Fatalf("rows = %d, want 1", n)
			}
			wantImported, wantSkipped := 0.0, 1.0
			if source == "" {
				wantImported, wantSkipped = 1, 0
			}
			if out["imported"] != wantImported || out["skipped"] != wantSkipped {
				t.Fatalf("response = %v", out)
			}
		})
	}
}
