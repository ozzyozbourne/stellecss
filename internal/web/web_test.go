package web

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ozzyozbourne/stellecss/internal/config"
)

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stellar.config.json")
	h, err := Handler(path, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, path
}

func TestEverySectionPageRenders(t *testing.T) {
	srv, _ := newTestServer(t)
	for _, g := range groups {
		for _, sec := range g.Sections {
			res, err := http.Get(srv.URL + sec.URL())
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `id="stellar-css"`) {
				t.Errorf("%s: status %d, body %.200s", sec.URL(), res.StatusCode, body)
			}
		}
	}
}

var linkRe = regexp.MustCompile(`/assets/css/stellar/([0-9a-f]+)`)

// A posted change must reach the config file, the served stylesheet and every
// page listening on /live-refresh, which is the whole point of the editor.
func TestPostedChangeRewritesConfigAndPushesNewStylesheet(t *testing.T) {
	srv, path := newTestServer(t)

	res, err := http.Get(srv.URL + "/live-refresh")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	links := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if m := linkRe.FindStringSubmatch(sc.Text()); m != nil {
				links <- m[1]
			}
		}
	}()
	first := <-links

	c := config.Default()
	c.General.Size.BaseMax = 2
	body, _ := json.Marshal(map[string]any{"cfg": toTree(c)})
	post, err := http.Post(srv.URL+"/general/size", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	reply, _ := io.ReadAll(post.Body)
	post.Body.Close()
	if !strings.Contains(string(reply), `role="alert"></output>`) {
		t.Fatalf("unexpected reply: %.300s", reply)
	}

	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.General.Size.BaseMax != 2 {
		t.Errorf("config file baseMax = %v, want 2", saved.General.Size.BaseMax)
	}

	var next string
	select {
	case next = <-links:
	case <-time.After(5 * time.Second):
		t.Fatal("no stylesheet pushed on /live-refresh")
	}
	if next == first {
		t.Fatalf("pushed hash %s did not change", next)
	}
	css, err := http.Get(srv.URL + "/assets/css/stellar/" + next)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(css.Body)
	css.Body.Close()
	// Step 0 of the size scale now tops out at the new base max.
	if !strings.Contains(string(b), "--size-0: clamp(0.5rem, ") || !strings.Contains(string(b), ", 2rem);") {
		t.Errorf("served CSS does not reflect baseMax 2")
	}
}

func TestInvalidChangeIsRejectedAndKeepsConfig(t *testing.T) {
	srv, path := newTestServer(t)
	c := config.Default()
	c.General.Viewport.Min = 2000 // above max
	body, _ := json.Marshal(map[string]any{"cfg": toTree(c)})
	post, err := http.Post(srv.URL+"/general/viewport", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	reply, _ := io.ReadAll(post.Body)
	post.Body.Close()
	if !strings.Contains(string(reply), "must be below max") {
		t.Errorf("reply has no validation error: %.300s", reply)
	}
	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.General.Viewport.Min != config.Default().General.Viewport.Min {
		t.Errorf("invalid change was saved")
	}
}
