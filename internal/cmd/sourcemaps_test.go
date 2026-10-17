package cmd

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/flaggr-dev/flaggr-cli/internal/api"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindSourceMapsFilesEachMapUnderItsMinifiedPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "chunks/app/page-3f9a.js.map"), "{}")
	writeFile(t, filepath.Join(dir, "chunks/main.js.map"), "{}")
	writeFile(t, filepath.Join(dir, "chunks/main.js"), "code")
	writeFile(t, filepath.Join(dir, "node_modules/dep/x.js.map"), "{}")
	single := filepath.Join(t.TempDir(), "server.js.map")
	writeFile(t, single, "{}")

	got, err := findSourceMaps([]string{dir, single}, "/_next/static/")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/_next/static/chunks/app/page-3f9a.js", "/_next/static/chunks/main.js", "/_next/static/server.js"}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range want {
		if got[i].File != w {
			t.Errorf("[%d] = %q, want %q", i, got[i].File, w)
		}
	}
	if none, _ := findSourceMaps([]string{dir}, ""); none[0].File != "/chunks/app/page-3f9a.js" {
		t.Errorf("without a prefix: %q", none[0].File)
	}
}

func TestUploadSourceMapSendsTheMapGzippedWithItsTarget(t *testing.T) {
	var mu sync.Mutex
	var got struct {
		method, path, file, release, encoding, auth, body string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("not gzip: %v", err)
			return
		}
		body, _ := io.ReadAll(zr)
		mu.Lock()
		got.method, got.path, got.encoding, got.auth, got.body = r.Method, r.URL.Path, r.Header.Get("Content-Encoding"), r.Header.Get("Authorization"), string(body)
		got.file, got.release = r.URL.Query().Get("file"), r.URL.Query().Get("release")
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"release":"web@1","file":"/a.js","bytes":2,"replaced":false}`))
	}))
	defer srv.Close()

	p := filepath.Join(t.TempDir(), "a.js.map")
	writeFile(t, p, `{"version":3}`)
	gz, err := gzipFile(p)
	if err != nil {
		t.Fatal(err)
	}
	client := api.NewRESTClient(srv.URL, "fgp_token")
	res, err := client.UploadSourceMap(t.Context(), "proj 1", "web@1", "/a.js", gz)
	if err != nil {
		t.Fatal(err)
	}
	if got.method != http.MethodPut || got.path != "/api/projects/proj 1/sourcemaps" || got.file != "/a.js" || got.release != "web@1" ||
		got.encoding != "gzip" || got.auth != "Bearer fgp_token" || got.body != `{"version":3}` || res.File != "/a.js" {
		t.Errorf("request = %+v, result = %+v", got, res)
	}
	if !bytes.HasPrefix(gz, []byte{0x1f, 0x8b}) {
		t.Error("gzipFile should gzip")
	}
}
