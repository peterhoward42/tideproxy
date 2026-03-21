package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
)

// IncomingRequest is the validated input payload for tide queries (latitude and longitude).
type IncomingRequest struct {
	Lat float64
	Lon float64
}

var (
	errTrailingJSON    = errors.New("trailing data after JSON value")
	errMissingLat      = errors.New("missing required field lat")
	errMissingLon      = errors.New("missing required field lon")
	errLatOutOfRange   = errors.New("lat must be in [-90, 90]")
	errLonOutOfRange   = errors.New("lon must be in [-180, 180]")
	errLatLonNotFinite = errors.New("lat and lon must be finite numbers")
)

// UnmarshalIncomingRequest parses JSON from data into an [IncomingRequest] and validates it.
// The JSON object must contain only "lat" and "lon" (no unknown fields), both required finite numbers
// with lat in [-90, 90] and lon in [-180, 180].
func UnmarshalIncomingRequest(data []byte) (IncomingRequest, error) {
	var raw struct {
		Lat *float64 `json:"lat"`
		Lon *float64 `json:"lon"`
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return IncomingRequest{}, err
	}

	var sink json.RawMessage
	if err := dec.Decode(&sink); err != io.EOF {
		if err == nil {
			return IncomingRequest{}, errTrailingJSON
		}
		return IncomingRequest{}, err
	}

	if raw.Lat == nil {
		return IncomingRequest{}, errMissingLat
	}
	if raw.Lon == nil {
		return IncomingRequest{}, errMissingLon
	}
	if !isFiniteCoord(*raw.Lat, *raw.Lon) {
		return IncomingRequest{}, errLatLonNotFinite
	}
	if *raw.Lat < -90 || *raw.Lat > 90 {
		return IncomingRequest{}, errLatOutOfRange
	}
	if *raw.Lon < -180 || *raw.Lon > 180 {
		return IncomingRequest{}, errLonOutOfRange
	}

	return IncomingRequest{Lat: *raw.Lat, Lon: *raw.Lon}, nil
}

func isFiniteCoord(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsInf(lat, 0) && !math.IsNaN(lon) && !math.IsInf(lon, 0)
}
