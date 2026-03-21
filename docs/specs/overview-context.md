# This is an overview specification for a proxy API

- The specification will be used to generate the code required using AI
- The specification defines an overview of the system required, to provide high level context for you
- You must not start generating code from this specification alone because we will do that incrementally in a set of smaller steps

## Role of the system
- To act as a cut-down, proxy server to the https://www.worldtides.info/ api

## High level architecture

- The system is a Google Cloud Function implemented with the Go coding language (version 1.25) and the
  Google Functions Framework package, using the "source deployment" option.

# Proxy API specification

## Endpoint
`GET /v1/tides`

## Query Parameters
- `lat` (number, required): latitude in [-90, 90]
- `lon` (number, required): longitude in [-180, 180]

## Behaviour
- Returns high and low tide extremes only
- Datum is fixed to Chart Datum (`CD`)
- Times are returned in UTC
- Coverage window:
  - Start: 00:00:00 UTC today
  - End: 00:00:00 UTC three days later (exclusive)
- No caching
- Upstream attribution is included

## Response (200)
```json
{
  "tides": [
    {
      "type": "High",
      "time": "2026-03-21T06:12:00Z",
      "heightMetres": 4.81
    }
  ],
  "datum": "CD",
  "windowStart": "2026-03-21T00:00:00Z",
  "expiresAt": "2026-03-24T00:00:00Z",
  "attribution": "Tidal predictions covered by various copyrights."
}
```

### Fields
- `tides`: array of tide extremes sorted by time
  - `type`: `"High"` or `"Low"`
  - `time`: ISO 8601 UTC timestamp
  - `heightMetres`: height in metres relative to CD
- `datum`: always `"CD"`
- `windowStart`: inclusive UTC start of forecast window
- `expiresAt`: exclusive UTC end of forecast window
- `attribution`: upstream copyright string

## Errors

### 400 Bad Request
Invalid or missing query parameters.

```json
{
  "error": {
    "code": "INVALID_QUERY",
    "message": "..."
  }
}
```

### 502 Bad Gateway
Upstream request failed.

```json
{
  "error": {
    "code": "UPSTREAM_ERROR",
    "message": "Failed to retrieve tidal data"
  }
}
```

# Implementation

- The proxy API must:
	- Receive GET requests as specified above and formulate onward requests requests to the https://www.worldtides.info/extremes endpoint
	- Formulate responses in the shape described above and reply with them
	- Obtain the API key required for the worldtides API request from a configuration setting on the Goggle Cloud Function deployment.