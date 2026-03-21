package app

import (
	"errors"
	"time"
)

var errNilIncomingResponse = errors.New("incoming response is nil")

// TidesAPIExtreme is one high or low tide in the proxy API success payload
// (docs/specs/overview.md).
type TidesAPIExtreme struct {
	Type         string    `json:"type"`
	Time         time.Time `json:"time"`
	HeightMetres float64   `json:"heightMetres"`
}

// TidesAPIResponse is the GET /v1/tides success JSON shape (overview.md).
type TidesAPIResponse struct {
	Tides       []TidesAPIExtreme `json:"tides"`
	Datum       string            `json:"datum"`
	WindowStart time.Time         `json:"windowStart"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	Attribution string            `json:"attribution"`
}

// SynthesiseTidesAPIResponse maps a validated [IncomingResponse] into the proxy
// API success model. The coverage window matches [SynthesiseOutputRequest]:
// UTC midnight at the start of the calendar day of at, for outputWindowDays
// full days; expiresAt is the exclusive end instant.
func SynthesiseTidesAPIResponse(in *IncomingResponse, at time.Time) (*TidesAPIResponse, error) {
	if in == nil {
		return nil, errNilIncomingResponse
	}

	now := at.UTC()
	windowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	expiresAt := windowStart.Add(time.Duration(outputWindowSeconds) * time.Second)

	tides := make([]TidesAPIExtreme, len(in.Extremes))
	for i := range in.Extremes {
		ex := &in.Extremes[i]
		tides[i] = TidesAPIExtreme{
			Type:         ex.Type,
			Time:         ex.Time.UTC(),
			HeightMetres: ex.HeightMetres,
		}
	}

	return &TidesAPIResponse{
		Tides:       tides,
		Datum:       chartDatum,
		WindowStart: windowStart,
		ExpiresAt:   expiresAt,
		Attribution: in.Copyright,
	}, nil
}
