package crash

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/flaggr-dev/flaggr-cli/internal/config"
)

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestEnabled(t *testing.T) {
	on, off := true, false
	cases := []struct {
		name string
		cfg  *config.Config
		vars map[string]string
		want bool
	}{
		{"off by default", &config.Config{}, nil, false},
		{"config on", &config.Config{CrashReports: &on}, nil, true},
		{"config off", &config.Config{CrashReports: &off}, nil, false},
		{"env on beats config off", &config.Config{CrashReports: &off}, map[string]string{"FLAGGR_CRASH_REPORTS": "1"}, true},
		{"env off beats config on", &config.Config{CrashReports: &on}, map[string]string{"FLAGGR_CRASH_REPORTS": "off"}, false},
		{"DO_NOT_TRACK beats everything", &config.Config{CrashReports: &on}, map[string]string{"DO_NOT_TRACK": "1", "FLAGGR_CRASH_REPORTS": "1"}, false},
		{"CI beats everything", &config.Config{CrashReports: &on}, map[string]string{"CI": "true"}, false},
		{"CI=false doesn't count", &config.Config{CrashReports: &on}, map[string]string{"CI": "false"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := Enabled(c.cfg, env(c.vars))
			if got != c.want || reason == "" {
				t.Errorf("Enabled = %v (%q), want %v", got, reason, c.want)
			}
		})
	}
}

type ingest struct {
	mu     sync.Mutex
	auth   []string
	events []map[string]any
}

func newIngest(t *testing.T) (*ingest, string) {
	in := &ingest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Events []map[string]any `json:"events"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		in.mu.Lock()
		in.auth = append(in.auth, r.Header.Get("Authorization"))
		in.events = append(in.events, body.Events...)
		in.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)
	return in, srv.URL + "/api/ingest/cli-crashes"
}

func crashWith(g Guard) (repanicked any) {
	defer func() { repanicked = recover() }()
	defer g.Recover()
	var items []int
	_ = items[3] // index out of range
	return nil
}

func TestGuardReportsWhenOnAndRepanics(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no config file: the env decides
	in, url := newIngest(t)
	var stderr bytes.Buffer
	value := crashWith(Guard{
		Version:   "v0.6.0",
		Command:   func() string { return "flags list" },
		Stderr:    &stderr,
		Getenv:    env(map[string]string{"FLAGGR_CRASH_REPORTS": "1"}),
		IngestURL: url,
	})
	if value == nil {
		t.Fatal("the panic was swallowed")
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	if len(in.events) != 1 {
		t.Fatalf("events = %d", len(in.events))
	}
	e := in.events[0]
	if in.auth[0] != "" {
		t.Errorf("a crash report carries no credential, got %q", in.auth[0])
	}
	data := e["data"].(map[string]any)
	if e["release"] != "flaggr-cli@0.6.0" || e["environment"] != "cli" || data["command"] != "flags list" || data["os"] == nil || data["level"] != "fatal" {
		t.Errorf("event = %v", e)
	}
	if !strings.Contains(e["message"].(string), "index out of range") {
		t.Errorf("message = %v", e["message"])
	}
	if !strings.Contains(stderr.String(), "crash report was sent") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestGuardOnlyHintsWhenOff(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	in, url := newIngest(t)
	var stderr bytes.Buffer
	if crashWith(Guard{Version: "0.6.0", Stderr: &stderr, Getenv: env(nil), IngestURL: url}) == nil {
		t.Fatal("the panic was swallowed")
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	if len(in.events) != 0 {
		t.Errorf("sent %d events with reports off", len(in.events))
	}
	if !strings.Contains(stderr.String(), "flaggr crash-reports on") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestGuardDoesNothingWithoutAPanic(t *testing.T) {
	func() {
		defer Guard{Getenv: env(map[string]string{"FLAGGR_CRASH_REPORTS": "1"}), IngestURL: "http://127.0.0.1:1"}.Recover()
	}()
}
