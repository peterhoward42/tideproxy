package app

import (
	"errors"
	"time"
)

const (
	worldTidesHTTPSHost = "www.worldtides.info"
	worldTidesAPIv3Path = "/api/v3"
	chartDatum          = "CD"
	heightUnitsMeters   = "meters"
	// outputWindowDays is the number of full UTC calendar days covered by the
	// extremes request: [today 00:00 UTC, today+outputWindowDays 00:00 UTC).
	outputWindowDays    = 3
	outputWindowSeconds = outputWindowDays * 24 * 60 * 60
)

var errNilIncomingRequest = errors.New("incoming request is nil")

// OutputRequest is the logical description of a WorldTides API v3 request for
// high and low tide extremes. It is not an [net/http.Request]; callers combine
// these fields with deployment configuration (for example the API key) when
// building the outbound HTTP call.
//
// The time window matches the proxy specification: UTC midnight at the start of
// the current calendar day, for outputWindowDays full days. WorldTides supports
// a date parameter anchored to local midnight at the coordinates; this proxy
// instead uses StartUnix and LengthSeconds so the window stays aligned to UTC.
type OutputRequest struct {
	Scheme        string
	Host          string
	Path          string
	Lat           float64
	Lon           float64
	Datum         string
	Units         string
	Extremes      bool
	StartUnix     int64
	LengthSeconds int64
}

// SynthesiseOutputRequest maps a validated [IncomingRequest] into an
// [OutputRequest] for the WorldTides extremes endpoint. The coverage interval
// is anchored to the current UTC date at the time of the call.
func SynthesiseOutputRequest(in *IncomingRequest) (*OutputRequest, error) {
	if in == nil {
		return nil, errNilIncomingRequest
	}
	if !isFiniteCoord(in.Lat, in.Lon) {
		return nil, errLatLonNotFinite
	}
	if in.Lat < -90 || in.Lat > 90 {
		return nil, errLatOutOfRange
	}
	if in.Lon < -180 || in.Lon > 180 {
		return nil, errLonOutOfRange
	}

	now := time.Now().UTC()
	windowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return &OutputRequest{
		Scheme:        "https",
		Host:          worldTidesHTTPSHost,
		Path:          worldTidesAPIv3Path,
		Lat:           in.Lat,
		Lon:           in.Lon,
		Datum:         chartDatum,
		Units:         heightUnitsMeters,
		Extremes:      true,
		StartUnix:     windowStart.Unix(),
		LengthSeconds: outputWindowSeconds,
	}, nil
}
