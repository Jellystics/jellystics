package pluginsync

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

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
	advance := fakeClock(s)

	runOK(t, s)
	st, _ := s.Status(context.Background())
	if st.MeLinked || st.MeLinkError == nil || st.LastError != nil || !st.Healthy {
		t.Fatalf("status after failed link = %+v", st)
	}
	advance(31 * time.Second)
	runOK(t, s) // retried after the backoff
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

// fakeClock makes s.now controllable; the returned func advances it.
func fakeClock(s *Service) func(time.Duration) {
	now := time.Now()
	s.now = func() time.Time { return now }
	return func(d time.Duration) { now = now.Add(d) }
}

func TestLinkRepushTriggers(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false); f.reportHeld = true })
	s, db := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"
	advance := fakeClock(s)
	n := func() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.links) }

	runOK(t, s)
	if n() != 1 {
		t.Fatalf("initial link: %d", n())
	}
	runOK(t, s)
	if n() != 1 {
		t.Fatalf("re-pushed while plugin holds our url + secret: %d", n())
	}

	// Plugin holds another secret (restored config, other instance).
	f.set(func(f *fakePlugin) { f.meFingerprint = sptr("0000000000000000") })
	runOK(t, s)
	if n() != 2 {
		t.Fatalf("fingerprint mismatch not re-pushed: %d", n())
	}

	// Plugin holds another url (Jellystics state says it pushed ours).
	f.set(func(f *fakePlugin) { f.meLinkedURL = sptr("http://old:3000") })
	runOK(t, s)
	if n() != 3 {
		t.Fatalf("url mismatch not re-pushed: %d", n())
	}

	// secretMismatch: rate-limited (last push just now), then re-pushed.
	f.set(func(f *fakePlugin) { f.meError = sptr(MeErrorSecretMismatch) })
	runOK(t, s)
	if n() != 3 {
		t.Fatalf("secretMismatch re-pushed too soon: %d", n())
	}
	if st, _ := s.Status(context.Background()); st.MeError == nil || *st.MeError != MeErrorSecretMismatch {
		t.Fatalf("meError not exposed: %+v", st.MeError)
	}
	advance(6 * time.Minute)
	runOK(t, s)
	if n() != 4 {
		t.Fatalf("secretMismatch not re-pushed: %d", n())
	}
	if st, _ := s.Status(context.Background()); st.MeError != nil {
		t.Fatalf("meError not cleared after the push: %v", *st.MeError)
	}

	// meLinked=false.
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false) })
	runOK(t, s)
	if n() != 5 {
		t.Fatalf("unlinked not re-pushed: %d", n())
	}

	// The plugin's report wins over our record: a matching fingerprint and
	// url mean up to date even if our stored hash differs.
	db.Exec(`UPDATE plugin_sync_state SET me_linked_secret_hash = 'x', me_linked_url = 'http://other'`)
	runOK(t, s)
	if n() != 5 {
		t.Fatalf("re-pushed although the plugin holds our url + secret: %d", n())
	}

	// Plugin disappears: link state reset (L7).
	f.set(func(f *fakePlugin) { f.absent = true })
	runOK(t, s)
	if st, _ := s.Status(context.Background()); st.MeLinked || st.MeReachable != nil || st.MeError != nil {
		t.Fatalf("link state not reset: %+v", st)
	}
}

func TestLinkIncompatiblePluginResetsLink(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false) })
	s, _ := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"
	runOK(t, s)
	f.set(func(f *fakePlugin) { f.schemaVersion = 99 })
	_, _ = s.Run(context.Background())
	if st, _ := s.Status(context.Background()); st.MeLinked || st.MeReachable != nil {
		t.Fatalf("link state not reset: %+v", st)
	}
}

func TestLinkErrorClearedWhenUpToDate(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false) })
	s, db := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"
	runOK(t, s)
	db.Exec(`UPDATE plugin_sync_state SET me_link_error = 'old dial timeout'`)
	runOK(t, s) // up to date: no push, error cleared
	st, _ := s.Status(context.Background())
	if len(f.links) != 1 || st.MeLinkError != nil {
		t.Fatalf("links %d, error %v", len(f.links), st.MeLinkError)
	}
}

func TestLinkBackoff(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.meLinked = bptr(false); f.linkFail = true })
	s, _ := newTestService(t, f)
	s.PublicURL = "http://jellystics.lan:3000"
	advance := fakeClock(s)
	n := func() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.links) }

	runOK(t, s) // failure 1 → retry in 30 s
	advance(20 * time.Second)
	runOK(t, s)
	if n() != 1 {
		t.Fatalf("retried during backoff: %d", n())
	}
	advance(11 * time.Second)
	runOK(t, s) // failure 2 → retry in 60 s
	if n() != 2 {
		t.Fatalf("not retried after 30 s: %d", n())
	}
	advance(59 * time.Second)
	runOK(t, s)
	if n() != 2 {
		t.Fatalf("retried before 60 s: %d", n())
	}
	advance(2 * time.Second)
	runOK(t, s) // failure 3
	if n() != 3 {
		t.Fatalf("not retried after 60 s: %d", n())
	}
	if st, _ := s.Status(context.Background()); st.MeLinkError == nil || st.LastError != nil || !st.Healthy {
		t.Fatalf("status = %+v", st)
	}
	// Recovery resets the backoff.
	f.set(func(f *fakePlugin) { f.linkFail = false })
	advance(2 * time.Minute)
	runOK(t, s)
	st, _ := s.Status(context.Background())
	if n() != 4 || !st.MeLinked || st.MeLinkError != nil || s.linkFailures != 0 {
		t.Fatalf("links %d, status %+v", n(), st)
	}
	for i, want := range map[int]time.Duration{1: 30 * time.Second, 2: time.Minute, 3: 2 * time.Minute, 7: 30 * time.Minute, 20: 30 * time.Minute} {
		if got := linkBackoff(i); got != want {
			t.Errorf("linkBackoff(%d) = %s, want %s", i, got, want)
		}
	}
}

func TestStatusDoesNotDial(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	db.Exec(`UPDATE app_config SET "JF_HOST" = 'http://10.255.255.1:1' WHERE "ID" = 1`)
	start := time.Now()
	st, err := s.Status(context.Background())
	if err != nil || time.Since(start) > time.Second || st.MeURL != "" || st.MeURLSource != MeURLSourceAuto {
		t.Fatalf("status %+v err %v in %s", st, err, time.Since(start))
	}
}

func TestPreferUsableIP(t *testing.T) {
	v4 := func() (net.IP, error) { return net.ParseIP("192.168.1.20"), nil }
	none := func() (net.IP, error) { return nil, context.DeadlineExceeded }
	cases := []struct {
		in   string
		v4   func() (net.IP, error)
		want string
	}{
		{"10.0.0.2", v4, "10.0.0.2"},
		{"fd00::5", v4, "fd00::5"},
		{"fe80::1", v4, "192.168.1.20"},
		{"::1", v4, "192.168.1.20"},
		{"fe80::1", none, "fe80::1"},
	}
	for _, tc := range cases {
		if got := jellyfin.PreferUsableIP(net.ParseIP(tc.in), tc.v4).String(); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.in, got, tc.want)
		}
	}
}
