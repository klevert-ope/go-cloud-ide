package apperr

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type Kind string

const (
	KindInvalid  Kind = "invalid"
	KindNotFound Kind = "not_found"
	KindConflict Kind = "conflict"
	KindExternal Kind = "external"
	KindInternal Kind = "internal"
	KindMethod   Kind = "method_not_allowed"
)

type Error struct {
	Op      string
	Kind    Kind
	Message string
	Err     error
}

// Error formats the application error with its operation context when available.
func (e *Error) Error() string {
	switch {
	case e == nil:
		return "<nil>"
	case e.Op != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	case e.Op != "":
		return e.Op
	case e.Err != nil:
		return e.Err.Error()
	default:
		return e.Message
	}
}

// Unwrap exposes the wrapped error for errors.Is and errors.As checks.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// E wraps an error with application metadata while preserving existing app errors.
func E(op string, kind Kind, message string, err error) error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		if appErr.Op == "" {
			appErr.Op = op
		}
		if appErr.Kind == "" {
			appErr.Kind = kind
		}
		if appErr.Message == "" {
			appErr.Message = message
		}
		return appErr
	}

	return &Error{
		Op:      op,
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}

// New creates an application error without an underlying cause.
func New(op string, kind Kind, message string) error {
	return &Error{
		Op:      op,
		Kind:    kind,
		Message: message,
	}
}

// KindOf extracts the application error kind and defaults to internal errors.
func KindOf(err error) Kind {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Kind != "" {
		return appErr.Kind
	}

	return KindInternal
}

// MessageOf returns a safe client-facing error message.
func MessageOf(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}

	return "internal server error"
}

// StatusCode maps an application error kind to an HTTP status code.
func StatusCode(err error) int {
	switch KindOf(err) {
	case KindInvalid:
		return http.StatusBadRequest
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindMethod:
		return http.StatusMethodNotAllowed
	case KindExternal:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// Write logs an error and sends the appropriate HTTP response format back to the client.
func Write(w http.ResponseWriter, r *http.Request, err error) {
	status := StatusCode(err)
	message := MessageOf(err)

	log.Printf("request failed: method=%s path=%s status=%d err=%v", r.Method, r.URL.Path, status, err)

	if r.Header.Get("HX-Request") == "true" {
		http.Error(w, message, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
