package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bqua0007/kanban-server/internal/kanban"
)

const (
	// maxStateBytes bounds the body of PUT /api/state, which carries a whole board set.
	maxStateBytes = 16 << 20
	// maxBodyBytes bounds every other request body.
	maxBodyBytes = 1 << 20
)

// writeJSON sends v as the JSON body of a response. The any parameter is the JSON boundary:
// whatever encoding/json can encode is accepted.
func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		// A value that cannot be encoded is a bug in the server, not a bad request.
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// A failed Write means the client has gone away; there is nobody left to tell.
	_, _ = w.Write(append(data, '\n'))
}

// writeError sends {"error": message} with the given status.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data, _ := json.Marshal(map[string]string{"error": message}) // a map of strings always encodes
	_, _ = w.Write(append(data, '\n'))
}

// requestError is an error that already knows which status code to send.
type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string { return e.message }

func badRequest(format string, args ...any) error {
	return &requestError{status: http.StatusBadRequest, message: fmt.Sprintf(format, args...)}
}

// decodeBody reads one JSON value from the request body into dst. It rejects bodies over limit
// bytes, empty bodies, unknown fields, and anything after the JSON value.
func decodeBody(w http.ResponseWriter, r *http.Request, limit int64, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return describeDecodeError(err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return describeDecodeError(err)
		}
		return badRequest("unexpected data after the JSON value")
	}
	return nil
}

// describeDecodeError turns the many ways decoding can fail into a client-facing error.
func describeDecodeError(err error) error {
	var (
		tooLarge  *http.MaxBytesError
		syntax    *json.SyntaxError
		wrongType *json.UnmarshalTypeError
	)
	switch {
	case errors.Is(err, io.EOF):
		return badRequest("the request body is empty")
	case errors.As(err, &tooLarge):
		return &requestError{
			status:  http.StatusRequestEntityTooLarge,
			message: fmt.Sprintf("the request body is too large (limit %d bytes)", tooLarge.Limit),
		}
	case errors.As(err, &syntax), errors.Is(err, io.ErrUnexpectedEOF):
		return badRequest("the request body is not valid JSON")
	case errors.As(err, &wrongType):
		return badRequest("the field %q has the wrong type: want %s", wrongType.Field, wrongType.Type)
	default:
		// Unknown fields and the errors from State's own decoder end up here; their messages
		// already say what is wrong, for example: json: unknown field "extra".
		return badRequest("%s", err.Error())
	}
}

// fail writes the response for an error returned by decoding or by the store. Errors the
// client can fix get their own message and a fitting status; anything else is logged and
// reported as a plain 500 so internal details do not leak.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var reqErr *requestError
	switch {
	case errors.As(err, &reqErr):
		writeError(w, reqErr.status, reqErr.message)
	case errors.Is(err, kanban.ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, kanban.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, kanban.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
