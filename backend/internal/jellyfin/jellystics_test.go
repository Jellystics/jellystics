package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetJellysticsInfo_Absent(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	info, err := c.GetJellysticsInfo(context.Background())
	if err != nil || info != nil {
		t.Fatalf("absent plugin: info=%v err=%v, want nil, nil", info, err)
	}
}

func TestGetJellysticsInfo_ErrorAndSuccess(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := c.GetJellysticsInfo(context.Background()); err == nil {
		t.Fatal("500 must be an error")
	}

	var gotAuth string
	c, _ = newTestClient(t, "secret", func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/Jellystics/Info" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"pluginVersion":"1.2.0.0","schemaVersion":1,"cursor":42,
			"firstPlayAt":"2026-01-02T03:04:05.0000000+00:00",
			"counts":{"plays":10,"libraryEvents":2,"userDataEvents":3,"logins":4},"serverTimeZone":"Europe/Paris"}`))
	})
	info, err := c.GetJellysticsInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotAuth, `Token="secret"`) {
		t.Errorf("auth header = %q", gotAuth)
	}
	if info.PluginVersion != "1.2.0.0" || info.SchemaVersion != 1 || info.Cursor != 42 ||
		info.Counts.Plays != 10 || info.Counts.Logins != 4 || info.ServerTimeZone != "Europe/Paris" ||
		info.FirstPlayAt == nil || info.FirstPlayAt.Year() != 2026 {
		t.Fatalf("info = %+v", info)
	}
}

func TestGetJellysticsPage_Plays(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Jellystics/Plays" || r.URL.Query().Get("since") != "7" || r.URL.Query().Get("limit") != "500" {
			t.Errorf("url = %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"p1","userId":"u1","itemId":"i1","itemType":"Movie",
			"startedAt":"2026-11-22T10:00:00+00:00","endedAt":null,"watchedMs":45000,
			"streamChanges":"[{\"kind\":\"audio\",\"index\":2}]","transcodeReasons":"[\"VideoCodecNotSupported\"]",
			"framerate":23.976,"state":"open","lastEventAt":"2026-11-22T10:01:00Z","source":"plugin","version":8}],
			"next":8,"hasMore":true}`))
	})
	page, err := c.GetJellysticsPage(context.Background(), JellysticsPlays, 7, 500)
	if err != nil {
		t.Fatal(err)
	}
	if page.Next != 8 || !page.HasMore {
		t.Fatalf("page = %+v", page)
	}
	plays, err := page.Plays()
	if err != nil || len(plays) != 1 {
		t.Fatalf("plays = %v, %v", plays, err)
	}
	p := plays[0]
	if p.Id != "p1" || p.WatchedMs != 45000 || p.EndedAt != nil || p.State != "open" || p.Version != 8 ||
		!p.StartedAt.Equal(time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("play = %+v", p)
	}
	if string(p.TranscodeReasons) != `["VideoCodecNotSupported"]` || !json.Valid(p.StreamChanges) {
		t.Fatalf("embedded json: reasons=%s changes=%s", p.TranscodeReasons, p.StreamChanges)
	}
}

func TestGetJellysticsPage_OtherTables(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Jellystics/LibraryEvents":
			_, _ = w.Write([]byte(`{"items":[{"id":"l1","kind":"added","at":"2026-11-22T10:00:00Z","itemId":"i","sizeBytes":12,"version":1}],"next":1,"hasMore":false}`))
		case "/Jellystics/UserDataEvents":
			_, _ = w.Write([]byte(`{"items":[{"id":"d1","kind":"played","at":"2026-11-22T10:00:00Z","reason":"TogglePlayed","version":2}],"next":2,"hasMore":false}`))
		case "/Jellystics/Logins":
			_, _ = w.Write([]byte(`{"items":[{"id":"g1","at":"2026-11-22T10:00:00Z","success":true,"version":3}],"next":3,"hasMore":false}`))
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	p, err := c.GetJellysticsPage(ctx, JellysticsLibraryEvents, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	lib, _ := p.LibraryEvents()
	if len(lib) != 1 || lib[0].Kind != "added" || *lib[0].SizeBytes != 12 {
		t.Fatalf("library = %+v", lib)
	}
	p, _ = c.GetJellysticsPage(ctx, JellysticsUserDataEvents, 0, 0)
	ud, _ := p.UserDataEvents()
	if len(ud) != 1 || *ud[0].Reason != "TogglePlayed" {
		t.Fatalf("user data = %+v", ud)
	}
	p, _ = c.GetJellysticsPage(ctx, JellysticsLogins, 0, 0)
	lg, _ := p.Logins()
	if len(lg) != 1 || !lg[0].Success || lg[0].Version != 3 {
		t.Fatalf("logins = %+v", lg)
	}
	if _, err := c.GetJellysticsPage(ctx, "Nope", 0, 0); err == nil {
		t.Fatal("unknown table must fail")
	}
}

func TestPostJellysticsPlays(t *testing.T) {
	var body struct {
		Items []map[string]any `json:"items"`
	}
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Jellystics/Import/Plays" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	})
	play := JellysticsPlay{Id: "x", UserId: "u", ItemId: "i", StartedAt: FlexTime{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		TranscodeReasons: EmbeddedJSON(`["A"]`), Source: "import-jellystics"}
	if err := c.PostJellysticsPlays(context.Background(), []JellysticsPlay{play}); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0]["id"] != "x" || body.Items[0]["transcodeReasons"] != `["A"]` ||
		body.Items[0]["startedAt"] != "2026-01-01T00:00:00Z" {
		t.Fatalf("body = %+v", body.Items)
	}
	if err := c.PostJellysticsPlays(context.Background(), make([]JellysticsPlay, 501)); err == nil {
		t.Fatal("more than 500 plays must be refused")
	}
}
