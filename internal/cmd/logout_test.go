package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaggr-dev/flaggr-cli/internal/config"
)

func TestLogoutNoticeSaysWhereToRevokeTheToken(t *testing.T) {
	pat := "fgp_" + strings.Repeat("ab", 32)
	cases := []struct {
		name    string
		token   string
		apiURL  string
		want    []string
		wantNot []string
	}{
		{
			name:    "personal access token",
			token:   pat,
			apiURL:  "https://flaggr.example/",
			want:    []string{"personal access token itself stays valid", "Profile → Personal access tokens (https://flaggr.example/console/profile#personal-access-tokens)"},
			wantNot: []string{pat, "FLAGGR_API_TOKEN"},
		},
		{
			name:    "project API token",
			token:   "fgr_project_token",
			apiURL:  "https://flaggr.example",
			want:    []string{"project API token itself stays valid until you revoke it", "Settings → API tokens (https://flaggr.example/console)"},
			wantNot: []string{"fgr_project_token"},
		},
		{
			name:   "other token, default URL",
			token:  "eyJhbGciOi.jwt.sig",
			apiURL: "",
			want:   []string{"stays valid until it expires or you revoke it", "https://flaggr.dev/console/profile#personal-access-tokens"},
		},
	}
	for _, tc := range cases {
		got := strings.Join(logoutNotice(tc.token, tc.apiURL, ""), "\n")
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: notice %q lacks %q", tc.name, got, w)
			}
		}
		for _, w := range tc.wantNot {
			if strings.Contains(got, w) {
				t.Errorf("%s: notice %q contains %q", tc.name, got, w)
			}
		}
	}

	if lines := logoutNotice("", "https://flaggr.example", ""); len(lines) != 0 {
		t.Errorf("nothing to revoke without a token, got %q", lines)
	}
	if got := strings.Join(logoutNotice(pat, "https://flaggr.example", pat), "\n"); !strings.Contains(got, "FLAGGR_API_TOKEN is still set") {
		t.Errorf("notice %q doesn't mention FLAGGR_API_TOKEN", got)
	}
}

// captureStdout returns what fn printed to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	return <-done
}

func TestLogoutRemovesTheConfigAndSaysTheTokenStaysValid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FLAGGR_API_URL", "")
	t.Setenv("FLAGGR_API_TOKEN", "")
	// Never touch the real ~/.flaggr: the config must resolve inside the temp home.
	path := config.Path()
	if !strings.HasPrefix(path, home) {
		t.Fatalf("config path %q is outside the test's home %q", path, home)
	}

	pat := "fgp_" + strings.Repeat("cd", 32)
	if err := config.Save(&config.Config{APIURL: "https://flaggr.example", APIToken: pat}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cmd := newLogoutCmd()
	cmd.SetArgs([]string{})
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Errorf("logout: %v", err)
		}
	})

	if _, err := os.Stat(filepath.Join(home, ".flaggr", "config.json")); !os.IsNotExist(err) {
		t.Errorf("config still exists after logout (stat err %v)", err)
	}
	for _, want := range []string{"Logged out", "stays valid until it expires or you revoke it", "https://flaggr.example/console/profile#personal-access-tokens"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	}
	if strings.Contains(out, pat) {
		t.Errorf("output echoes the token: %q", out)
	}
}
