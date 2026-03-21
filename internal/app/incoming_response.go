package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"time"
)

// IncomingResponse is a validated WorldTides API v3 payload containing tide
// extremes and attribution, suitable for mapping onto the proxy API shape in
// docs/specs/overview.md.
type IncomingResponse struct {
	Copyright string
	Extremes  []IncomingExtreme
}

// IncomingExtreme is one high or low tide extreme from the upstream extremes list.
type IncomingExtreme struct {
	Type         string
	Time         time.Time
	HeightMetres float64
}

var (
	errIncomingResponseTrailingJSON   = errors.New("trailing data after JSON value")
	errIncomingResponseStatus         = errors.New("upstream status must be 200")
	errIncomingResponseCopyright      = errors.New("missing or empty upstream copyright")
	errIncomingResponseDatum          = errors.New("upstream responseDatum must be CD")
	errIncomingResponseExtremes       = errors.New("upstream extremes must be present")
	errIncomingResponseExtremeDt      = errors.New("each extreme must include a dt unix timestamp")
	errIncomingResponseExtremeType    = errors.New("extreme type must be High or Low")
	errIncomingResponseExtremeHeight  = errors.New("extreme height must be a finite number")
	errIncomingResponseExtremesOrder  = errors.New("extremes must be sorted by time ascending")
)

// ParseIncomingResponse parses JSON from data into an [IncomingResponse] and
// validates fields required to build spec-compliant proxy responses: HTTP-level
// success is assumed already; JSON status 200, non-empty copyright, chart datum
// CD, and a well-formed extremes list sorted by time.
func ParseIncomingResponse(data []byte) (IncomingResponse, error) {
	var raw struct {
		Status         *int          `json:"status"`
		Copyright      *string       `json:"copyright"`
		ResponseDatum  *string       `json:"responseDatum"`
		Extremes       []rawExtreme  `json:"extremes"`
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&raw); err != nil {
		return IncomingResponse{}, err
	}

	var sink json.RawMessage
	if err := dec.Decode(&sink); err != io.EOF {
		if err == nil {
			return IncomingResponse{}, errIncomingResponseTrailingJSON
		}
		return IncomingResponse{}, err
	}

	if raw.Status == nil {
		return IncomingResponse{}, errIncomingResponseStatus
	}
	if *raw.Status != 200 {
		return IncomingResponse{}, errIncomingResponseStatus
	}
	if raw.Copyright == nil || *raw.Copyright == "" {
		return IncomingResponse{}, errIncomingResponseCopyright
	}
	if raw.ResponseDatum == nil || *raw.ResponseDatum != chartDatum {
		return IncomingResponse{}, errIncomingResponseDatum
	}
	// extremes omitted or JSON null: decoder leaves nil slice; reject.
	if raw.Extremes == nil {
		return IncomingResponse{}, errIncomingResponseExtremes
	}

	extremes := make([]IncomingExtreme, 0, len(raw.Extremes))
	for i := range raw.Extremes {
		ex := &raw.Extremes[i]
		if ex.Dt == nil {
			return IncomingResponse{}, errIncomingResponseExtremeDt
		}
		if ex.Type == nil {
			return IncomingResponse{}, errIncomingResponseExtremeType
		}
		switch *ex.Type {
		case "High", "Low":
		default:
			return IncomingResponse{}, errIncomingResponseExtremeType
		}
		if ex.Height == nil || !isFiniteFloat(*ex.Height) {
			return IncomingResponse{}, errIncomingResponseExtremeHeight
		}
		t := time.Unix(*ex.Dt, 0).UTC()
		extremes = append(extremes, IncomingExtreme{
			Type:         *ex.Type,
			Time:         t,
			HeightMetres: *ex.Height,
		})
	}

	if !extremesSortedByTime(extremes) {
		return IncomingResponse{}, errIncomingResponseExtremesOrder
	}

	return IncomingResponse{
		Copyright: *raw.Copyright,
		Extremes:  extremes,
	}, nil
}

type rawExtreme struct {
	Dt     *int64   `json:"dt"`
	Height *float64 `json:"height"`
	Type   *string  `json:"type"`
}

func extremesSortedByTime(extremes []IncomingExtreme) bool {
	for i := 1; i < len(extremes); i++ {
		if !extremes[i].Time.After(extremes[i-1].Time) {
			return false
		}
	}
	return true
}

func isFiniteFloat(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
