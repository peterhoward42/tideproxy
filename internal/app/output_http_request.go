package app

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

var (
	errNilOutputRequest         = errors.New("output request is nil")
	errEmptyAPIKey              = errors.New("api key is empty")
	errIncompleteOutputEndpoint = errors.New("output request has empty scheme, host, or path")
)

// BuildOutputHTTPRequest returns a GET request for the WorldTides API v3
// endpoint described by out. apiKey is sent as the required "key" query
// parameter.
func BuildOutputHTTPRequest(ctx context.Context, out *OutputRequest, apiKey string) (*http.Request, error) {
	if out == nil {
		return nil, errNilOutputRequest
	}
	if apiKey == "" {
		return nil, errEmptyAPIKey
	}
	if out.Scheme == "" || out.Host == "" || out.Path == "" {
		return nil, errIncompleteOutputEndpoint
	}

	q := url.Values{}
	q.Set("key", apiKey)
	q.Set("lat", strconv.FormatFloat(out.Lat, 'f', -1, 64))
	q.Set("lon", strconv.FormatFloat(out.Lon, 'f', -1, 64))
	q.Set("datum", out.Datum)
	q.Set("units", out.Units)
	if out.Extremes {
		q.Set("extremes", "")
	}
	q.Set("start", strconv.FormatInt(out.StartUnix, 10))
	q.Set("length", strconv.FormatInt(out.LengthSeconds, 10))

	u := url.URL{
		Scheme:   out.Scheme,
		Host:     out.Host,
		Path:     out.Path,
		RawQuery: q.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}
