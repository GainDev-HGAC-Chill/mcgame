package registration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	appauth "mcgame-server/internal/application/auth"
	pkgerrors "mcgame-server/pkg/errors"
)

type registerFunc func(context.Context, *appauth.WebRegistrationRequest) error

func (f registerFunc) RegisterWeb(ctx context.Context, req *appauth.WebRegistrationRequest) error {
	return f(ctx, req)
}

const validBody = `{"username":"1","password":"1","confirmPassword":"1"}`

func request(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "http://game.local:8888/api/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://game.local:8888")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	return r
}

func TestHandlerResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"created", nil, 201, "created"},
		{"duplicate", pkgerrors.Wrap(pkgerrors.ErrAlreadyExists, "race"), 409, "username_taken"},
		{"invalid", pkgerrors.ErrInvalidInput, 400, "invalid_input"},
		{"unavailable", errors.New("SQL secret failure"), 503, "unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(registerFunc(func(_ context.Context, req *appauth.WebRegistrationRequest) error {
				if req.Username != "1" || req.Password != "1" || req.ConfirmPassword != "1" {
					t.Fatal("input modified")
				}
				return tc.err
			}))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, request(validBody))
			var response struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || response.Code != tc.code {
				t.Fatalf("unexpected response %d %s", w.Code, response.Code)
			}
			if strings.Contains(w.Body.String(), "SQL") || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("unsafe response")
			}
			if len(h.permits) != 0 {
				t.Fatal("permit leaked")
			}
		})
	}
	w := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(w, request(validBody))
	if w.Code != 503 {
		t.Fatal("nil service not unavailable")
	}
}

func TestHandlerRejectsInvalidRequests(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		mutate     func(*http.Request)
		status     int
	}{
		{"wrong method", validBody, func(r *http.Request) { r.Method = "GET" }, 405},
		{"options", validBody, func(r *http.Request) { r.Method = "OPTIONS" }, 405},
		{"media type", validBody, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, 400},
		{"missing media type", validBody, func(r *http.Request) { r.Header.Del("Content-Type") }, 400},
		{"unknown", `{"username":"1","password":"1","confirmPassword":"1","email":""}`, nil, 400},
		{"missing", `{"username":"1","password":"1"}`, nil, 400},
		{"wrong case", `{"Username":"1","password":"1","confirmPassword":"1"}`, nil, 400},
		{"null", `{"username":null,"password":"1","confirmPassword":"1"}`, nil, 400},
		{"duplicate field", `{"username":"1","username":"2","password":"1","confirmPassword":"1"}`, nil, 400},
		{"trailing document", validBody + `{}`, nil, 400},
		{"trailing junk", validBody + `x`, nil, 400},
		{"malformed", `{`, nil, 400},
		{"array", `[]`, nil, 400},
		{"empty", "", nil, 400},
		{"oversize", validBody + strings.Repeat(" ", 4096), nil, 400},
		{"mismatch", `{"username":"1","password":"1","confirmPassword":"2"}`, nil, 400},
		{"admin", validBody, func(r *http.Request) { r.URL.Path = "/api/admin/players" }, 404},
		{"lookup", validBody, func(r *http.Request) { r.URL.Path = "/api/accounts" }, 404},
		{"trailing slash", validBody, func(r *http.Request) { r.URL.Path = "/api/register/" }, 404},
		{"encoded path", validBody, func(r *http.Request) { r.URL.RawPath = "/api/%72egister" }, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(registerFunc(func(context.Context, *appauth.WebRegistrationRequest) error {
				t.Fatal("invalid request reached service")
				return nil
			}))
			r := request(tc.body)
			if tc.mutate != nil {
				tc.mutate(r)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d", w.Code, tc.status)
			}
		})
	}
}

func TestHandlerUnicode(t *testing.T) {
	for _, field := range []string{"username", "password", "confirmPassword"} {
		for _, tc := range []struct {
			name, value string
			valid       bool
		}{
			{"raw invalid", "\xff", false},
			{"high surrogate", `\ud800`, false},
			{"low surrogate", `\udfff`, false},
			{"wrong pair", `\ud800\u0041`, false},
			{"emoji pair", `\ud83d\ude00`, true},
			{"replacement escaped", `\ufffd`, true},
			{"replacement literal", "�", true},
			{"literal emoji", "😀", true},
			{"escaped backslash", `\\ud800`, true},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				values := map[string]string{"username": "1", "password": "1", "confirmPassword": "1"}
				values[field] = tc.value
				unicodeBody := `{"username":"` + values["username"] + `","password":"` + values["password"] + `","confirmPassword":"` + values["confirmPassword"] + `"}`
				if validJSONUnicode([]byte(unicodeBody)) != tc.valid {
					t.Fatal("individual field Unicode decoding decision incorrect")
				}
				if field == "password" {
					values["confirmPassword"] = tc.value
				}
				if field == "confirmPassword" {
					values["password"] = tc.value
				}
				body := `{"username":"` + values["username"] + `","password":"` + values["password"] + `","confirmPassword":"` + values["confirmPassword"] + `"}`
				called := false
				h := NewHandler(registerFunc(func(context.Context, *appauth.WebRegistrationRequest) error { called = true; return nil }))
				w := httptest.NewRecorder()
				h.ServeHTTP(w, request(body))
				if called != tc.valid || (w.Code == 201) != tc.valid {
					t.Fatalf("valid=%v status=%d called=%v", tc.valid, w.Code, called)
				}
			})
		}
	}
}

func TestHandlerOrigins(t *testing.T) {
	for _, tc := range []struct {
		name, origin, site string
		valid              bool
	}{
		{"same", "http://game.local:8888", "same-origin", true},
		{"missing", "", "", true},
		{"missing origin same metadata", "", "same-origin", true},
		{"null", "null", "same-origin", false},
		{"wrong port", "http://game.local:8080", "same-origin", false},
		{"missing port", "http://game.local", "same-origin", false},
		{"scheme", "https://game.local:8888", "same-origin", false},
		{"suffix", "http://game.local.attacker:8888", "same-origin", false},
		{"userinfo", "http://game.local:8888@attacker", "same-origin", false},
		{"path", "http://game.local:8888/path", "same-origin", false},
		{"query", "http://game.local:8888?q=1", "same-origin", false},
		{"fragment", "http://game.local:8888#x", "same-origin", false},
		{"cross-site", "http://game.local:8888", "cross-site", false},
		{"same-site", "http://game.local:8888", "same-site", false},
		{"missing origin cross-site", "", "cross-site", false},
		{"opaque metadata", "", "none", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(validBody)
			r.Header.Del("Origin")
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			r.Header.Set("Sec-Fetch-Site", tc.site)
			r.Header.Set("X-Forwarded-Host", "attacker")
			r.Header.Set("X-Forwarded-Proto", "https")
			if sameOrigin(r) != tc.valid {
				t.Fatal("unexpected origin decision")
			}
			called := false
			h := NewHandler(registerFunc(func(context.Context, *appauth.WebRegistrationRequest) error { called = true; return nil }))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if called != tc.valid || (w.Code == 201) != tc.valid {
				t.Fatal("origin not enforced at handler")
			}
		})
	}
	r := request(validBody)
	r.Header.Add("Origin", "http://attacker")
	if sameOrigin(r) {
		t.Fatal("multiple origins accepted")
	}
	r = request(validBody)
	r.Header.Set("Origin", "http://attacker")
	r.Header.Set("X-Forwarded-Host", "attacker")
	if sameOrigin(r) {
		t.Fatal("forwarded host trusted")
	}
	r = request(validBody)
	r.TLS = &tls.ConnectionState{}
	r.Header.Set("Origin", "https://game.local:8888")
	if !sameOrigin(r) {
		t.Fatal("same-origin TLS request rejected")
	}
}

func TestHandlerConcurrencyAndCancellation(t *testing.T) {
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	h := NewHandler(registerFunc(func(context.Context, *appauth.WebRegistrationRequest) error {
		entered <- struct{}{}
		<-release
		return errors.New("unavailable")
	}))
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); h.ServeHTTP(httptest.NewRecorder(), request(validBody).WithContext(ctx)) }()
	}
	for i := 0; i < 4; i++ {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			close(release)
			wg.Wait()
			t.Fatal("registrations did not enter")
		}
	}
	cancel()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request(validBody))
	if w.Code != 429 || len(h.permits) != 4 {
		close(release)
		wg.Wait()
		t.Fatal("cancelled requests released active hashing permits")
	}
	close(release)
	wg.Wait()
	if len(h.permits) != 0 {
		t.Fatal("permits leaked after hashing returns")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request(validBody))
	if w.Code != 503 {
		t.Fatal("permit not reusable after error")
	}
}

func TestHandlerExactBodyCeiling(t *testing.T) {
	body := validBody + strings.Repeat(" ", 4096-len(validBody))
	h := NewHandler(registerFunc(func(context.Context, *appauth.WebRegistrationRequest) error { return nil }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request(body))
	if w.Code != 201 {
		t.Fatal("4 KiB body rejected")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request(body+" "))
	if w.Code != 400 {
		t.Fatal("body over 4 KiB accepted")
	}
}
