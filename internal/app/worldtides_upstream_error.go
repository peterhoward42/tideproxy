package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// WorldTidesUpstreamError is the JSON envelope WorldTides returns when a request
// fails. Confirmed via live probe: invalid API key yields HTTP 400 and
// {"status":400,"error":"API key is invalid"} with matching HTTP and JSON status.
// Credit exhaustion is not documented publicly; proxy detection treats any
// non-empty upstream error containing "credit" (case-insensitive) as exhaustion.
type WorldTidesUpstreamError struct {
	Status int
	Error  string
}

var (
	errWorldTidesUpstreamErrorStatus = errors.New("upstream status is required")
	errWorldTidesUpstreamErrorText   = errors.New("upstream error is required")
)

// ParseWorldTidesUpstreamError decodes a WorldTides failure JSON body into
// [WorldTidesUpstreamError].
func ParseWorldTidesUpstreamError(data []byte) (WorldTidesUpstreamError, error) {
	var raw struct {
		Status *int    `json:"status"`
		Error  *string `json:"error"`
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&raw); err != nil {
		return WorldTidesUpstreamError{}, err
	}

	var sink json.RawMessage
	if err := dec.Decode(&sink); err != io.EOF {
		if err == nil {
			return WorldTidesUpstreamError{}, errIncomingResponseTrailingJSON
		}
		return WorldTidesUpstreamError{}, err
	}

	if raw.Status == nil {
		return WorldTidesUpstreamError{}, errWorldTidesUpstreamErrorStatus
	}
	if raw.Error == nil || *raw.Error == "" {
		return WorldTidesUpstreamError{}, errWorldTidesUpstreamErrorText
	}

	return WorldTidesUpstreamError{
		Status: *raw.Status,
		Error:  *raw.Error,
	}, nil
}

const worldTidesErrorInvalidAPIKey = "API key is invalid"

// CreditsExhausted reports whether the upstream error indicates operator quota
// exhaustion (substring "credit", case-insensitive).
func (e WorldTidesUpstreamError) CreditsExhausted() bool {
	return strings.Contains(strings.ToLower(e.Error), "credit")
}

// InvalidAPIKey reports whether the upstream error indicates a misconfigured
// operator API key.
func (e WorldTidesUpstreamError) InvalidAPIKey() bool {
	return e.Error == worldTidesErrorInvalidAPIKey
}

// ProxyErrorForWorldTidesUpstream maps a parsed upstream failure to proxy API
// error response fields per docs/specs/overview.md.
func ProxyErrorForWorldTidesUpstream(e WorldTidesUpstreamError) (httpStatus int, code, message string) {
	switch {
	case e.CreditsExhausted():
		return http.StatusServiceUnavailable, "UPSTREAM_CREDITS_EXHAUSTED", "Monthly API credits exhausted"
	case e.InvalidAPIKey():
		return http.StatusInternalServerError, "INTERNAL_ERROR", "WorldTides API key is invalid"
	default:
		return http.StatusBadGateway, "UPSTREAM_ERROR", "Failed to retrieve tidal data"
	}
}
