package pluginsync

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
)


func runOK(t *testing.T, s *Service) {
	t.Helper()
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestLinkPushedWhenNotLinked(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false) })
	s, db := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000/"

	runOK(t, s)
	if len(f.links) != 1 {
		t.Fatalf("links = %v", f.links)
	}
	var secret string
	db.Raw(`SELECT me_secret FROM app_config WHERE "ID" = 1`).Scan(&secret)
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(secret) {
		t.Fatalf("secret = %q", secret)
	}
	if got := f.links[0]; got["url"] != "http://jellystics.lan:3000" || got["secret"] != secret {
		t.Fatalf("link body = %v", got)
	}
	st, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.MeLinked || st.MeReachable == nil || !*st.MeReachable || st.MeURL != "http://jellystics.lan:3000" ||
		st.MeURLSource != MeURLSourceEnv || st.MeLinkedURL == nil || st.MeLinkError != nil {
		t.Fatalf("status = %+v", st)
	}

	// Unchanged url / secret, plugin linked: no new push.
	runOK(t, s)
	runOK(t, s)
	if len(f.links) != 1 {
		t.Fatalf("re-pushed while unchanged: %d", len(f.links))
	}

	// Url change: pushed again, same secret.
	s.PublicURL = "http://10.0.0.5:3000"
	runOK(t, s)
	if len(f.links) != 2 || f.links[1]["url"] != "http://10.0.0.5:3000" || f.links[1]["secret"] != secret {
		t.Fatalf("links = %v", f.links)
	}

	// Plugin lost its link (reinstall): pushed again.
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false) })
	runOK(t, s)
	if len(f.links) != 3 {
		t.Fatalf("not re-pushed after unlink: %d", len(f.links))
	}

	// Secret changed (e.g. restore): pushed again with the new secret.
	db.Exec(`UPDATE app_config SET me_secret = '' WHERE "ID" = 1`)
	runOK(t, s)
	if len(f.links) != 4 || f.links[3]["secret"] == secret || f.links[3]["secret"] == "" {
		t.Fatalf("not re-pushed after secret change: %v", f.links)
	}
}

func TestLinkUnsupportedPluginIgnored(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false); f.linkMissing = true })
	s, _ := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"

	runOK(t, s)
	runOK(t, s)
	f.set(func(f *fakePlugin) {
		if n := f.calls["/Jellystics/Link"]; n != 1 {
			t.Errorf("unsupported Link retried: %d calls", n)
		}
	})
	st, _ := s.Status(context.Background())
	if st.MeLinked || st.LastError != nil || st.MeLinkError != nil || !st.Healthy {
		t.Fatalf("status = %+v", st)
	}
}

func TestLinkSkippedForPluginWithoutFeature(t *testing.T) {
	f := newFakePlugin(t) // /Info without meLinked
	s, _ := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"
	runOK(t, s)
	if len(f.links) != 0 {
		t.Fatalf("pushed to a plugin without the feature: %v", f.links)
	}
}

func TestLinkFailureDoesNotFailSync(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false); f.failOn["/Jellystics/Link"] = 1 })
	s, _ := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"

	runOK(t, s)
	st, _ := s.Status(context.Background())
	if st.MeLinked || st.MeLinkError == nil || st.LastError != nil || !st.Healthy {
		t.Fatalf("status after failed link = %+v", st)
	}
	runOK(t, s) // retried next cycle
	st, _ = s.Status(context.Background())
	if len(f.links) != 1 || !st.MeLinked || st.MeLinkError != nil {
		t.Fatalf("links = %v, status = %+v", f.links, st)
	}
}

func TestLinkAutoURL(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false) })
	s, _ := newTestService(t, f)
	s.Port = "8123"
	runOK(t, s)
	if len(f.links) != 1 || f.links[0]["url"] != "http://127.0.0.1:8123" {
		t.Fatalf("links = %v", f.links)
	}
	st, _ := s.Status(context.Background())
	if st.MeURL != "http://127.0.0.1:8123" || st.MeURLSource != MeURLSourceAuto {
		t.Fatalf("status = %+v", st)
	}
}

func TestAutoMeURL(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	jf := jellyfin.NewClient(srv.URL, "")
	got, err := AutoMeURL(context.Background(), jf, "")
	if err != nil || got != "http://127.0.0.1:3000" {
		t.Fatalf("AutoMeURL = %q, %v", got, err)
	}
	if got := autoURL(net.ParseIP("fd00::5"), "3001"); got != "http://[fd00::5]:3001" {
		t.Fatalf("ipv6 = %q", got)
	}
	if _, err := AutoMeURL(context.Background(), jellyfin.NewClient("http://127.0.0.1:1", ""), "3000"); err == nil {
		t.Fatal("expected an error for an unreachable host")
	}
	u, err := ResolveMeURL(context.Background(), " http://x:1/ ", nil, "3000")
	if err != nil || u.URL != "http://x:1" || u.Source != MeURLSourceEnv {
		t.Fatalf("env = %+v, %v", u, err)
	}
}
