package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

// CreditsExhausted reports whether the upstream error indicates operator quota
// exhaustion (substring "credit", case-insensitive).
func (e WorldTidesUpstreamError) CreditsExhausted() bool {
	return strings.Contains(strings.ToLower(e.Error), "credit")
}
