package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func ctxFor(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/x?"+query, nil)
	return c
}

func TestParseV2(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("no tzdata")
	}
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, paris)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	utc := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// 2026-11-21 23:30 UTC is already 2026-11-22 00:30 in Paris.
	nowNearMidnight := utc("2026-11-21T23:30:00Z")

	cases := []struct {
		name             string
		query            string
		now              time.Time
		wantErr          bool
		from, to         time.Time
		prevFrom, prevTo time.Time
		page, size       int
		mediaType, typ   string
		desc             bool
	}{
		{name: "default uses local today", query: "", now: nowNearMidnight,
			from: at("2026-10-24 00:00"), to: at("2026-11-23 00:00"), page: 1, size: 25},
		{name: "explicit range, to exclusive", query: "from=2026-11-01&to=2026-11-08", now: nowNearMidnight,
			from: at("2026-11-01 00:00"), to: at("2026-11-08 00:00"), page: 1, size: 25},
		{name: "only to", query: "to=2026-11-10", now: nowNearMidnight,
			from: at("2026-10-11 00:00"), to: at("2026-11-10 00:00"), page: 1, size: 25},
		{name: "DST spring forward day is 23h", query: "from=2026-03-29&to=2026-03-30&compare=1", now: nowNearMidnight,
			from: utc("2026-03-28T23:00:00Z"), to: utc("2026-03-29T22:00:00Z"),
			prevFrom: utc("2026-03-27T23:00:00Z"), prevTo: utc("2026-03-28T23:00:00Z"), page: 1, size: 25},
		{name: "compare across DST fall back", query: "from=2026-10-25&to=2026-11-01&compare=true", now: nowNearMidnight,
			from: at("2026-10-25 00:00"), to: at("2026-11-01 00:00"),
			prevFrom: at("2026-10-18 00:00"), prevTo: at("2026-10-25 00:00"), page: 1, size: 25},
		{name: "paging, type and sort", query: "page=3&pageSize=500&type=music&sort=startedAt&desc=1&q=%20abc%20", now: nowNearMidnight,
			from: at("2026-10-24 00:00"), to: at("2026-11-23 00:00"), page: 3, size: 200, mediaType: "Audio", typ: "music", desc: true},
		{name: "series type", query: "type=series", now: nowNearMidnight,
			from: at("2026-10-24 00:00"), to: at("2026-11-23 00:00"), page: 1, size: 25, mediaType: "Episode", typ: "series"},
		{name: "from equals to", query: "from=2026-11-01&to=2026-11-01", wantErr: true},
		{name: "from after to", query: "from=2026-11-02&to=2026-11-01", wantErr: true},
		{name: "impossible date", query: "from=2026-02-30&to=2026-03-01", wantErr: true},
		{name: "bad format", query: "from=2026-1-1", wantErr: true},
		{name: "rfc3339 refused", query: "from=2026-11-01T00:00:00Z", wantErr: true},
		{name: "bad type", query: "type=book", wantErr: true},
		{name: "bad page", query: "page=0", wantErr: true},
		{name: "bad pageSize", query: "pageSize=abc", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := tc.now
			if now.IsZero() {
				now = nowNearMidnight
			}
			p, err := parseV2At(ctxFor(tc.query), now)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", p)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !p.Range.From.Equal(tc.from) || !p.Range.To.Equal(tc.to) {
				t.Errorf("range = %v .. %v, want %v .. %v", p.Range.From, p.Range.To, tc.from, tc.to)
			}
			if tc.prevFrom.IsZero() {
				if p.Prev != nil {
					t.Errorf("unexpected prev %+v", p.Prev)
				}
			} else if p.Prev == nil || !p.Prev.From.Equal(tc.prevFrom) || !p.Prev.To.Equal(tc.prevTo) {
				t.Errorf("prev = %+v, want %v .. %v", p.Prev, tc.prevFrom, tc.prevTo)
			}
			if p.Page != tc.page || p.PageSize != tc.size || p.MediaType != tc.mediaType || p.Type != tc.typ || p.Desc != tc.desc {
				t.Errorf("got %+v", p)
			}
		})
	}
}

func TestParseV2Trims(t *testing.T) {
	p, err := parseV2At(ctxFor("q=%20abc%20&userId=u1&libraryId=l1"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if p.Q != "abc" || p.UserID != "u1" || p.LibraryID != "l1" {
		t.Errorf("got %+v", p)
	}
}
