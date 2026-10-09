package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewLoginStateIsRandomBase64URL(t *testing.T) {
	a, err := newLoginState()
	if err != nil {
		t.Fatalf("newLoginState: %v", err)
	}
	b, err := newLoginState()
	if err != nil {
		t.Fatalf("newLoginState: %v", err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).MatchString(a) {
		t.Errorf("state %q is not 32 bytes of unpadded base64url", a)
	}
	if a == b {
		t.Error("two logins got the same state")
	}
}

func TestLoginStateMatches(t *testing.T) {
	state, _ := newLoginState()
	other, _ := newLoginState()
	cases := []struct {
		name     string
		expected string
		received string
		want     bool
	}{
		{"same", state, state, true},
		{"missing", state, "", false},
		{"different", state, other, false},
		{"prefix", state, state[:10], false},
		{"both empty", "", "", false},
	}
	for _, tc := range cases {
		if got := loginStateMatches(tc.expected, tc.received); got != tc.want {
			t.Errorf("%s: loginStateMatches = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCliAuthURLCarriesPortAndState(t *testing.T) {
	raw := cliAuthURL("https://flaggr.dev/", 54321, "st_ate-1")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	if u.Scheme+"://"+u.Host+u.Path != "https://flaggr.dev/auth/cli" {
		t.Errorf("auth URL %q, want https://flaggr.dev/auth/cli", raw)
	}
	if got := u.Query().Get("port"); got != "54321" {
		t.Errorf("port = %q, want 54321", got)
	}
	if got := u.Query().Get("state"); got != "st_ate-1" {
		t.Errorf("state = %q, want st_ate-1", got)
	}
	// /auth/cli only hands a token to a CLI that reads it from a POST body.
	if got := u.Query().Get("response_mode"); got != "form_post" {
		t.Errorf("response_mode = %q, want form_post", got)
	}
}

// serveCallback POSTs form to /callback the way /auth/cli's form does.
func serveCallback(h http.Handler, form url.Values) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	return rec
}

func serveCallbackGet(h http.Handler, query url.Values) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/callback?"+query.Encode(), nil))
	return rec
}

func noResult(t *testing.T, results <-chan callbackResult) {
	t.Helper()
	select {
	case r := <-results:
		t.Fatalf("a callback was delivered: %+v", r)
	default:
	}
}

// recv returns the delivered callback, failing (not hanging) when there is none.
func recv(t *testing.T, results <-chan callbackResult) callbackResult {
	t.Helper()
	select {
	case r := <-results:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("no callback was delivered")
		return callbackResult{}
	}
}

// pat-clients-4: a token in the callback URL lands in browser history, so the
// CLI never takes one from the query string.
func TestCallbackRefusesATokenInTheURL(t *testing.T) {
	state, _ := newLoginState()

	// Without the state: ignored, and the login keeps waiting.
	results := make(chan callbackResult, 1)
	h := newCallbackHandler(state, results)
	for _, q := range []url.Values{
		{"token": {"fgp_attacker"}},
		{"token": {"fgp_attacker"}, "state": {"not-the-state"}},
		{"state": {state}},
		{"error": {"access_denied"}, "state": {state}},
	} {
		rec := serveCallbackGet(h, q)
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodPost {
			t.Errorf("GET %v: status %d Allow %q, want 405 POST", q, rec.Code, rec.Header().Get("Allow"))
		}
	}
	noResult(t, results)

	// With the state: our own (older) sign-in page — the login fails and the
	// token isn't kept.
	rec := serveCallbackGet(h, url.Values{"token": {"fgp_in_history"}, "project_id": {"proj-1"}, "state": {state}})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET with token and state: status %d, want 400", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "fgp_in_history") {
		t.Error("the page echoed the token")
	}
	r := recv(t, results)
	if r.Token != "" || r.ProjectID != "" || r.Error != tokenInURLError {
		t.Errorf("delivered %+v, want only the token-in-URL error", r)
	}
	if !strings.Contains(r.Error, "Revoke it") {
		t.Errorf("error %q doesn't tell the user to revoke the token", r.Error)
	}

	// The login is over: a later POST gets a 409.
	if rec := serveCallback(h, url.Values{"token": {"fgp_real"}, "state": {state}}); rec.Code != http.StatusConflict {
		t.Errorf("POST after the failed login: status %d, want 409", rec.Code)
	}
}

func TestCallbackReadsOnlyThePostedForm(t *testing.T) {
	state, _ := newLoginState()
	results := make(chan callbackResult, 1)
	h := newCallbackHandler(state, results)

	// The state (and token) in the query of a POST don't count.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/callback?"+url.Values{"state": {state}, "token": {"fgp_query"}}.Encode(),
		strings.NewReader(url.Values{"token": {"fgp_body"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("state only in the query: status %d, want 400", rec.Code)
	}
	noResult(t, results)

	// A body over the limit is refused without delivering anything.
	big := url.Values{"token": {strings.Repeat("a", maxCallbackBodyBytes)}, "state": {state}}
	if rec := serveCallback(h, big); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized body: status %d, want 400", rec.Code)
	}
	noResult(t, results)

	// The posted form wins over anything in the query.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/callback?token=fgp_query",
		strings.NewReader(url.Values{"token": {"fgp_body"}, "state": {state}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("matching POST: status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if r := recv(t, results); r.Token != "fgp_body" {
		t.Errorf("delivered %+v, want the posted token", r)
	}
}

func TestCallbackIgnoresRequestsWithoutTheState(t *testing.T) {
	state, _ := newLoginState()
	results := make(chan callbackResult, 1)
	h := newCallbackHandler(state, results)

	for _, q := range []url.Values{
		{"token": {"fgr_attacker"}, "project_id": {"evil"}},
		{"token": {"fgr_attacker"}, "state": {"not-the-state"}},
		{"error": {"access_denied"}},
	} {
		rec := serveCallback(h, q)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%v: status %d, want 400", q, rec.Code)
		}
	}
	select {
	case r := <-results:
		t.Fatalf("a callback without the state was delivered: %+v", r)
	default:
	}

	// The real callback still completes the login afterwards.
	rec := serveCallback(h, url.Values{"token": {"fgp_real"}, "project_id": {"proj-1"}, "state": {state}})
	if rec.Code != http.StatusOK {
		t.Fatalf("matching callback: status %d, want 200", rec.Code)
	}
	r := recv(t, results)
	if r.Token != "fgp_real" || r.ProjectID != "proj-1" || r.Error != "" {
		t.Errorf("delivered %+v", r)
	}
}

func TestCallbackErrorWithStateFailsTheLoginAndIsEscaped(t *testing.T) {
	state, _ := newLoginState()
	results := make(chan callbackResult, 1)
	h := newCallbackHandler(state, results)

	rec := serveCallback(h, url.Values{"error": {"<script>alert(1)</script>"}, "state": {state}})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Error("error message reflected unescaped")
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("escaped error message missing from the page")
	}
	if r := recv(t, results); r.Error != "<script>alert(1)</script>" || r.Token != "" {
		t.Errorf("delivered %+v", r)
	}
}

func TestCallbackWithoutTokenFails(t *testing.T) {
	state, _ := newLoginState()
	results := make(chan callbackResult, 1)
	rec := serveCallback(newCallbackHandler(state, results), url.Values{"state": {state}})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", rec.Code)
	}
	if r := recv(t, results); r.Error != "no token received" {
		t.Errorf("delivered %+v", r)
	}
}

func TestSecondMatchingCallbackDoesNotBlock(t *testing.T) {
	state, _ := newLoginState()
	results := make(chan callbackResult, 1)
	h := newCallbackHandler(state, results)

	first := serveCallback(h, url.Values{"token": {"fgr_one"}, "state": {state}})
	if first.Code != http.StatusOK {
		t.Fatalf("first callback: status %d", first.Code)
	}

	done := make(chan int, 1)
	go func() { done <- serveCallback(h, url.Values{"token": {"fgr_two"}, "state": {state}}).Code }()
	select {
	case code := <-done:
		if code != http.StatusConflict {
			t.Errorf("second callback: status %d, want 409", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second callback blocked on the full results channel")
	}
	if r := recv(t, results); r.Token != "fgr_one" {
		t.Errorf("delivered %+v, want the first callback", r)
	}
}
