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
	// utcWindowLookbackDays is how many full UTC calendar days before today's
	// midnight are included in the coverage interval.
	utcWindowLookbackDays = 1
	// utcWindowEndOffsetFromToday is added to today's UTC midnight to get the
	// exclusive end instant (unchanged from the original three-day-forward spec).
	utcWindowEndOffsetFromToday = 3
)

var errNilIncomingRequest = errors.New("incoming request is nil")

// OutputRequest is the logical description of a WorldTides API v3 request for
// high and low tide extremes. It is not an [net/http.Request]; callers combine
// these fields with deployment configuration (for example the API key) when
// building the outbound HTTP call.
//
// The time window matches the proxy specification: from UTC midnight at the start
// of the calendar day before the current day, through UTC midnight three calendar
// days after today (exclusive). WorldTides supports a date parameter anchored to
// local midnight at the coordinates; this proxy instead uses StartUnix and
// LengthSeconds so the window stays aligned to UTC.
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

// utcTidesCoverageWindow returns the inclusive start and exclusive end of the
// proxy tide coverage interval for reference time at (docs/specs/overview.md).
func utcTidesCoverageWindow(at time.Time) (windowStart, expiresAt time.Time) {
	now := at.UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	windowStart = todayStart.AddDate(0, 0, -utcWindowLookbackDays)
	expiresAt = todayStart.AddDate(0, 0, utcWindowEndOffsetFromToday)
	return windowStart, expiresAt
}

// SynthesiseOutputRequest maps a validated [IncomingRequest] into an
// [OutputRequest] for the WorldTides extremes endpoint. The coverage interval
// follows [utcTidesCoverageWindow] for at.
func SynthesiseOutputRequest(in *IncomingRequest, at time.Time) (*OutputRequest, error) {
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

	windowStart, expiresAt := utcTidesCoverageWindow(at)
	lengthSeconds := int64(expiresAt.Sub(windowStart) / time.Second)

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
		LengthSeconds: lengthSeconds,
	}, nil
}
