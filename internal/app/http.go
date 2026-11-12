package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mindflow/internal/authentication"
	"mindflow/internal/fault"
	"net"
	"net/http"
	"net/url"
	"time"
)

type apiError struct {
	Status        int
	Code, Message string
}

// Error returns the HTTP failure message.
func (e *apiError) Error() string                    { return e.Message }
func problem(status int, code, message string) error { return &apiError{status, code, message} }
func bad(message string) error                       { return fault.Invalid(message) }
func randomID() string {
	buffer := make([]byte, identifierBytes)
	_, readErr := rand.Read(buffer)
	if readErr != nil {
		panic(readErr)
	}

	return hex.EncodeToString(buffer)
}

type rateWindow struct {
	Start time.Time
	Count int
}

func who(r *http.Request) authentication.Principal { return authentication.Current(r.Context()) }
func decode(w http.ResponseWriter, request *http.Request, value any) error {
	mediaType, _, parseError := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if parseError != nil || mediaType != "application/json" {
		return problem(http.StatusUnsupportedMediaType, "content_type", "Expected application/json.")
	}
	request.Body = http.MaxBytesReader(w, request.Body, jsonRequestBytes)
	document := json.NewDecoder(request.Body)
	document.DisallowUnknownFields()
	err := document.Decode(value)
	if err != nil {
		return bad("Invalid request body.")
	}
	if document.Decode(new(any)) != io.EOF {
		return bad("Expected one JSON value.")
	}

	return nil
}

func (a *App) endpoint(pattern string, auth, admin bool, fnValue func(http.ResponseWriter, *http.Request) (any, error)) {
	var handler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			err := a.checkOrigin(request)
			if err != nil {
				a.fail(writer, request, err)

				return
			}
		}
		response, err := fnValue(writer, request)
		if err != nil {
			a.fail(writer, request, err)

			return
		}
		if _, handled := response.(responseWritten); !handled && response != nil {
			writeJSON(writer, http.StatusOK, response)
		}
	})
	if admin {
		handler = authentication.RequireAdministrator(a.fail)(handler)
	}
	if auth {
		handler = a.authentication.Middleware(a.fail)(handler)
	}
	a.Mux.Handle(pattern, handler)
}

func (a *App) fail(writer http.ResponseWriter, request *http.Request, err error) {
	var aeValue *apiError
	if de, ok := errors.AsType[*fault.Error](err); ok {
		aeValue = &apiError{Status: statusFor(de.Code), Code: de.Code, Message: de.Message}
	}
	if aeValue == nil && !errors.As(err, &aeValue) {
		aeValue = &apiError{http.StatusInternalServerError, "server_error", "The operation could not be completed. Please retry."}
		a.Log.Error("request failed", "request", request.Header.Get("X-Request-ID"), "error", err)
	}
	writeJSON(writer, aeValue.Status, map[string]any{"code": aeValue.Code, "message": aeValue.Message, "requestID": request.Header.Get("X-Request-ID")})
}

func statusFor(code string) int {
	switch code {
	case "not_found":
		return http.StatusNotFound
	case "session_expired", "invalid_login":
		return http.StatusUnauthorized
	case "bootstrap", "forbidden", csrfKey, "origin":
		return http.StatusForbidden
	case "initialized", "email_exists", "metadata_conflict", "lease_lost", "revision_conflict":
		return http.StatusConflict
	case "document_limit", "upload_limit":
		return http.StatusRequestEntityTooLarge
	case "rate_limit":
		return http.StatusTooManyRequests
	default:
		return http.StatusBadRequest
	}
}

func (a *App) checkOrigin(request *http.Request) error {
	if request.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return problem(http.StatusForbidden, "origin", "Cross-site requests are not allowed.")
	}
	origin := request.Header.Get("Origin")
	if origin == "" {
		return nil
	}
	s, err := a.authentication.Settings(request.Context())
	if err != nil {
		return err
	}
	if origin == s.PublicURL {
		return nil
	}
	if a.Config.Dev && (origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" || origin == "http://"+request.Host) {
		return nil
	}
	// Before setup, the browser must be on the same public host.
	initialized, err := a.authentication.Initialized(request.Context())
	if err != nil {
		return err
	}
	u, e := url.Parse(origin)
	if !initialized && e == nil && u.Host == request.Host && u.Scheme == "https" {
		return nil
	}

	return problem(http.StatusForbidden, "origin", "Request origin does not match the application URL.")
}

func (a *App) limited(r *http.Request, key string) bool {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host == "" {
		host = r.RemoteAddr
	}
	key += ":" + host
	a.limitMu.Lock()
	defer a.limitMu.Unlock()
	value := a.limits[key]
	if time.Since(value.Start) > time.Minute {
		value = rateWindow{Start: time.Now()}
	}
	value.Count++
	a.limits[key] = value
	if len(a.limits) > maxRateWindows {
		for k, v := range a.limits {
			if time.Since(v.Start) > time.Minute {
				delete(a.limits, k)
			}
		}
	}

	return value.Count > maxRequestsPerMinute
}

func (a *App) authenticate(request *http.Request) error {
	cookie, err := request.Cookie("mindflow_session")
	if err != nil {
		return fault.Fault("session_expired", "Please sign in.")
	}

	_, err = a.authentication.Authenticate(request.Context(), cookie.Value)

	return err
}

// responseWritten marks a body already written by a streaming or binary handler.
type responseWritten struct{}
