package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"time"

	"mindflow/internal/domain"
)

type principalKey struct{}
type apiError struct {
	Status        int
	Code, Message string
}

func (e *apiError) Error() string                    { return e.Message }
func problem(status int, code, message string) error { return &apiError{status, code, message} }
func bad(message string) error                       { return domain.Invalid(message) }
func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type rateWindow struct {
	Start time.Time
	Count int
}

func who(r *http.Request) principal { p, _ := r.Context().Value(principalKey{}).(principal); return p }
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	mediaType, _, parseError := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if parseError != nil || mediaType != "application/json" {
		return problem(415, "content_type", "Expected application/json.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 17<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return bad("Invalid request body.")
	}
	if d.Decode(new(any)) != io.EOF {
		return bad("Expected one JSON value.")
	}
	return nil
}
func (a *App) endpoint(pattern string, auth, admin bool, fn func(http.ResponseWriter, *http.Request) (any, error)) {
	a.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var p principal
		if auth {
			var err error
			p, err = a.authenticate(r)
			if err != nil {
				a.fail(w, r, err)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), principalKey{}, p))
			if admin && p.Role != "admin" {
				a.fail(w, r, problem(403, "forbidden", "Administrator access required."))
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if err := a.checkOrigin(r); err != nil {
				a.fail(w, r, err)
				return
			}
			if auth && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(p.CSRF)) != 1 {
				a.fail(w, r, problem(403, "csrf", "Refresh the page and try again."))
				return
			}
		}
		result, err := fn(w, r)
		if err != nil {
			a.fail(w, r, err)
			return
		}
		if result != nil {
			writeJSON(w, 200, result)
		}
	})
}
func (a *App) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apiError
	var de *domain.Error
	if errors.As(err, &de) {
		ae = &apiError{Status: statusFor(de.Code), Code: de.Code, Message: de.Message}
	}
	if ae == nil && !errors.As(err, &ae) {
		ae = &apiError{500, "server_error", "The operation could not be completed. Please retry."}
		a.Log.Error("request failed", "request", r.Header.Get("X-Request-ID"), "error", err)
	}
	writeJSON(w, ae.Status, map[string]any{"code": ae.Code, "message": ae.Message, "requestID": r.Header.Get("X-Request-ID")})
}
func statusFor(code string) int {
	switch code {
	case "not_found":
		return 404
	case "session_expired", "invalid_login":
		return 401
	case "bootstrap", "forbidden", "csrf", "origin":
		return 403
	case "initialized", "email_exists", "metadata_conflict", "lease_lost", "revision_conflict":
		return 409
	case "document_limit", "upload_limit":
		return 413
	case "rate_limit":
		return 429
	default:
		return 400
	}
}
func (a *App) checkOrigin(r *http.Request) error {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return problem(403, "origin", "Cross-site requests are not allowed.")
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return nil
	}
	s, err := a.service.Settings(r.Context())
	if err != nil {
		return err
	}
	if origin == s.PublicURL {
		return nil
	}
	if a.Config.Dev && (origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" || origin == "http://"+r.Host) {
		return nil
	}
	// Before setup, the browser must be on the same public host.
	initialized, err := a.service.Initialized(r.Context())
	if err != nil {
		return err
	}
	u, e := url.Parse(origin)
	if !initialized && e == nil && u.Host == r.Host && u.Scheme == "https" {
		return nil
	}
	return problem(403, "origin", "Request origin does not match the application URL.")
}
func (a *App) limited(r *http.Request, key string) bool {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host == "" {
		host = r.RemoteAddr
	}
	key += ":" + host
	a.limitMu.Lock()
	defer a.limitMu.Unlock()
	v := a.limits[key]
	if time.Since(v.Start) > time.Minute {
		v = rateWindow{Start: time.Now()}
	}
	v.Count++
	a.limits[key] = v
	if len(a.limits) > 10000 {
		for k, v := range a.limits {
			if time.Since(v.Start) > time.Minute {
				delete(a.limits, k)
			}
		}
	}
	return v.Count > 20
}
func (a *App) authenticate(r *http.Request) (principal, error) {
	cookie, err := r.Cookie("mindflow_session")
	if err != nil {
		return principal{}, domain.Fault("session_expired", "Please sign in.")
	}
	return a.service.Authenticate(r.Context(), cookie.Value)
}
