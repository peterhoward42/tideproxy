package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const telegramCreditsExhaustedAlert = "tideproxy: WorldTides monthly API credits exhausted"

var (
	errQueryLatMissing = errors.New("missing required query parameter lat")
	errQueryLonMissing = errors.New("missing required query parameter lon")
)

func (a *Application) refTime() time.Time {
	return a.deps.Clock.Now()
}

// handleTides serves GET /v1/tides by validating query parameters, building the
// WorldTides extremes request, and performing the outbound HTTP call. A
// successful upstream body is parsed as a WorldTides extremes payload, mapped
// to the proxy API success shape, and written as JSON.
func (a *Application) handleTides(w http.ResponseWriter, r *http.Request) {
	payload, err := latLonQueryJSON(r.URL.Query())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	in, err := UnmarshalIncomingRequest(payload)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	out, err := SynthesiseOutputRequest(&in, a.refTime())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	req, err := BuildOutputHTTPRequest(r.Context(), out, a.deps.WorldTidesAPIKey)
	if err != nil {
		if errors.Is(err, errEmptyAPIKey) {
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "WorldTides API key is not configured")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	resp, err := a.deps.HTTPClient.Do(req)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "Failed to retrieve tidal data")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "Failed to retrieve tidal data")
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		a.writeWorldTidesUpstreamFailure(r.Context(), w, body)
		return
	}

	inResp, err := ParseIncomingResponse(body)
	if err != nil {
		a.writeWorldTidesUpstreamFailure(r.Context(), w, body)
		return
	}

	apiResp, err := SynthesiseTidesAPIResponse(&inResp, a.refTime())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if err := WriteTidesAPIResponse(w, apiResp); err != nil {
		return
	}

	a.notifyTelegramLoadSuccess(r.Context(), in.Lat, in.Lon)
}

func latLonQueryJSON(q url.Values) ([]byte, error) {
	latStr, lonStr := q.Get("lat"), q.Get("lon")
	if latStr == "" {
		return nil, errQueryLatMissing
	}
	if lonStr == "" {
		return nil, errQueryLonMissing
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return nil, fmt.Errorf("lat is not a valid number: %w", err)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return nil, fmt.Errorf("lon is not a valid number: %w", err)
	}
	return json.Marshal(map[string]float64{"lat": lat, "lon": lon})
}

type apiErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Application) writeWorldTidesUpstreamFailure(ctx context.Context, w http.ResponseWriter, body []byte) {
	upstreamErr, err := ParseWorldTidesUpstreamError(body)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "UPSTREAM_ERROR", "Failed to retrieve tidal data")
		return
	}
	if upstreamErr.CreditsExhausted() {
		a.notifyTelegramCreditsExhausted(ctx)
	}
	status, code, message := ProxyErrorForWorldTidesUpstream(upstreamErr)
	writeAPIError(w, status, code, message)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	var body apiErrorResponse
	body.Error.Code = code
	body.Error.Message = message
	_ = json.NewEncoder(w).Encode(body)
}
