package cmd

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flaggr-dev/flaggr-cli/internal/config"
	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var token, apiURL string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate the CLI with your Flaggr account",
		Long: `Authenticate the CLI by logging in via your browser.

Opens your browser to sign in and select a project.
You can also pass a token directly with --token for non-interactive use.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := config.Load()
			if apiURL != "" {
				c.APIURL = apiURL
			}

			// Direct token mode (non-interactive)
			if token != "" {
				c.APIToken = token
				if err := config.Save(c); err != nil {
					return err
				}
				output.Success("Token saved to " + config.Path())
				return nil
			}

			// Browser-based login with localhost redirect
			return browserAuthFlow(c)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "API token (skip browser auth)")
	cmd.Flags().StringVar(&apiURL, "url", "", "Flaggr API URL (default: https://flaggr.dev)")

	return cmd
}

type callbackResult struct {
	Token     string `json:"token"`
	ProjectID string `json:"project_id"`
	Error     string `json:"error"`
}

// newLoginState returns a random state for one browser login. The /auth/cli
// page echoes it back on the localhost callback, so a web page that only
// knows (or guesses) the port can't complete — or cancel — the login with a
// token of its own choosing.
func newLoginState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// loginStateMatches compares a callback's state with this login's in
// constant time.
func loginStateMatches(expected, received string) bool {
	if expected == "" || received == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(received)) == 1
}

// formPostResponseMode asks /auth/cli to hand over the token as a form POST
// to the callback (OAuth's form_post response mode) instead of in the
// callback URL, which browser history, history sync and extensions keep.
const formPostResponseMode = "form_post"

// maxCallbackBodyBytes bounds the callback form: a token, a project id and
// the state.
const maxCallbackBodyBytes = 16 << 10

// tokenInURLError fails a login whose sign-in page put the token in the
// callback URL (a Flaggr server older than this CLI): the token is already in
// the browser's history, so it isn't saved.
const tokenInURLError = "the sign-in page sent the token in the URL, where your browser history keeps it, so it was not saved. " +
	"Revoke it in Flaggr (Profile → Personal access tokens, or the project's API tokens), then create a token there and run `flaggr login --token <token>`"

// cliAuthURL is the browser sign-in page for a login listening on port.
func cliAuthURL(apiURL string, port int, state string) string {
	q := url.Values{}
	q.Set("port", strconv.Itoa(port))
	q.Set("state", state)
	q.Set("response_mode", formPostResponseMode)
	return strings.TrimRight(apiURL, "/") + "/auth/cli?" + q.Encode()
}

// newCallbackHandler serves /callback. /auth/cli POSTs a form with the token,
// an optional project_id and this login's state; the URL query is never
// read for them. A request without this login's state gets a 400 and is
// otherwise ignored, so the login keeps waiting. The first matching callback
// is delivered on results (buffered, so it never blocks); later ones get a
// 409. A GET carrying a token is refused — and fails the login when it also
// carries the state, since only a sign-in page that knows the state (an
// older Flaggr server) sends one.
func newCallbackHandler(state string, results chan<- callbackResult) http.Handler {
	var once sync.Once
	deliver := func(result callbackResult) bool {
		delivered := false
		once.Do(func() {
			results <- result
			delivered = true
		})
		return delivered
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")

		if r.Method != http.MethodPost {
			q := r.URL.Query()
			if q.Get("token") != "" && loginStateMatches(state, q.Get("state")) {
				if !deliver(callbackResult{Error: tokenInURLError}) {
					w.WriteHeader(http.StatusConflict)
					fmt.Fprint(w, successPage("This sign-in has already finished. Return to your terminal.", false))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, successPage("Not saved: the token was sent in the page address, which your browser history keeps. Revoke it in Flaggr and see your terminal.", false))
				return
			}
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, successPage("This sign-in doesn't match the flaggr login waiting in your terminal, so it was ignored.", false))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxCallbackBodyBytes)
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, successPage("This sign-in couldn't be read, so it was ignored.", false))
			return
		}
		form := r.PostForm

		if !loginStateMatches(state, form.Get("state")) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, successPage("This sign-in doesn't match the flaggr login waiting in your terminal, so it was ignored.", false))
			return
		}

		result := callbackResult{Token: form.Get("token"), ProjectID: form.Get("project_id")}
		if errMsg := form.Get("error"); errMsg != "" {
			result = callbackResult{Error: errMsg}
		} else if result.Token == "" {
			result = callbackResult{Error: "no token received"}
		}

		if !deliver(result) {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, successPage("This sign-in has already finished. Return to your terminal.", false))
			return
		}

		if result.Error != "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, successPage("Authentication failed: "+result.Error, false))
			return
		}
		fmt.Fprint(w, successPage("You are now logged in. You can close this tab.", true))
	})
	return mux
}

func browserAuthFlow(cfg *config.Config) error {
	apiURL, _ := config.Resolve(cfg)

	state, err := newLoginState()
	if err != nil {
		return fmt.Errorf("failed to create login state: %w", err)
	}

	// Start a local HTTP server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	resultCh := make(chan callbackResult, 1)

	server := &http.Server{
		Handler:           newCallbackHandler(state, resultCh),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		_ = server.Serve(listener)
	}()

	// Open the browser to the auth page
	authURL := cliAuthURL(apiURL, port, state)
	fmt.Println("Your browser has been opened to visit:")
	fmt.Println()
	fmt.Printf("    %s\n", authURL)
	fmt.Println()

	if err := openBrowser(authURL); err != nil {
		fmt.Println("If your browser didn't open, visit the URL above manually.")
	}

	fmt.Println("Waiting for authentication...")

	// Wait for the callback or timeout
	select {
	case result := <-resultCh:
		// Shut down the local server
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)

		if result.Error != "" {
			return fmt.Errorf("authentication failed: %s", result.Error)
		}

		cfg.APIToken = result.Token
		if result.ProjectID != "" {
			cfg.DefaultProjectID = result.ProjectID
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Println()
		output.Success("You are now logged in!")
		fmt.Printf("  Credentials saved to %s\n", config.Path())
		if result.ProjectID != "" {
			fmt.Printf("  Default project: %s\n", result.ProjectID)
		}
		fmt.Println()
		fmt.Println("  Try: flaggr status")
		return nil

	case <-time.After(5 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		return fmt.Errorf("authentication timed out — run `flaggr login` again")
	}
}

func successPage(message string, success bool) string {
	color := "#ef4444"
	icon := "&#10007;"
	if success {
		color = "#22c55e"
		icon = "&#10003;"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Flaggr CLI</title></head>
<body style="font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #0a0a0a; color: #fafafa;">
  <div style="text-align: center;">
    <div style="font-size: 48px; color: %s; margin-bottom: 16px;">%s</div>
    <h2 style="margin: 0 0 8px;">%s</h2>
    <p style="color: #a1a1aa;">Return to your terminal.</p>
  </div>
</body>
</html>`, color, icon, html.EscapeString(message))
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
