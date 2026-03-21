package app

import (
	"encoding/json"
	"errors"
	"net/http"
)

var errNilTidesAPIResponse = errors.New("tides API response is nil")

// WriteTidesAPIResponse writes a 200 OK JSON body for GET /v1/tides using the
// proxy API success shape (overview.md).
func WriteTidesAPIResponse(w http.ResponseWriter, resp *TidesAPIResponse) error {
	if resp == nil {
		return errNilTidesAPIResponse
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(resp)
}
