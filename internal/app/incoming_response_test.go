package app

import (
	"errors"
	"io"
	"slices"
	"testing"
	"time"
)

func worldTidesUpstreamFixture(extremes []map[string]any) map[string]any {
	return map[string]any{
		"status":          200,
		"copyright":       "Tidal predictions covered by various copyrights.",
		"requestDatum":    "CD",
		"responseDatum":   "CD",
		"extremes":        extremes,
		"responseLat":     33.768321,
		"responseLon":     -118.195617,
	}
}

func TestParseIncomingResponse_valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
		want    IncomingResponse
	}{
		{
			name: "single high tide",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"dt": int64(1710994320), "date": "2024-03-21T06:12:00+0000", "height": 4.81, "type": "High"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			want: IncomingResponse{
				Copyright: "Tidal predictions covered by various copyrights.",
				Extremes: []IncomingExtreme{
					{Type: "High", Time: mustUnixUTC(t, 1710994320), HeightMetres: 4.81},
				},
			},
		},
		{
			name: "two extremes ascending",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"dt": int64(100), "height": 1.0, "type": "Low"},
					{"dt": int64(200), "height": 2.0, "type": "High"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			want: IncomingResponse{
				Copyright: "Tidal predictions covered by various copyrights.",
				Extremes: []IncomingExtreme{
					{Type: "Low", Time: mustUnixUTC(t, 100), HeightMetres: 1},
					{Type: "High", Time: mustUnixUTC(t, 200), HeightMetres: 2},
				},
			},
		},
		{
			name: "empty extremes list",
			payload: func(t *testing.T) []byte {
				return mustMarshalJSON(t, worldTidesUpstreamFixture([]map[string]any{}))
			},
			want: IncomingResponse{
				Copyright: "Tidal predictions covered by various copyrights.",
				Extremes:  []IncomingExtreme{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseIncomingResponse(tt.payload(t))
			if err != nil {
				t.Fatalf("ParseIncomingResponse: %v", err)
			}
			if got.Copyright != tt.want.Copyright {
				t.Fatalf("Copyright: got %q want %q", got.Copyright, tt.want.Copyright)
			}
			if len(got.Extremes) != len(tt.want.Extremes) {
				t.Fatalf("len(Extremes): got %d want %d", len(got.Extremes), len(tt.want.Extremes))
			}
			for i := range tt.want.Extremes {
				if got.Extremes[i].Type != tt.want.Extremes[i].Type {
					t.Fatalf("Extremes[%d].Type: got %q want %q", i, got.Extremes[i].Type, tt.want.Extremes[i].Type)
				}
				if !got.Extremes[i].Time.Equal(tt.want.Extremes[i].Time) {
					t.Fatalf("Extremes[%d].Time: got %v want %v", i, got.Extremes[i].Time, tt.want.Extremes[i].Time)
				}
				if got.Extremes[i].HeightMetres != tt.want.Extremes[i].HeightMetres {
					t.Fatalf("Extremes[%d].HeightMetres: got %v want %v", i, got.Extremes[i].HeightMetres, tt.want.Extremes[i].HeightMetres)
				}
			}
		})
	}
}

func mustUnixUTC(t *testing.T, sec int64) time.Time {
	t.Helper()
	return time.Unix(sec, 0).UTC()
}

func TestParseIncomingResponse_validationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
		wantErr error
	}{
		{
			name: "status not 200",
			payload: func(t *testing.T) []byte {
				m := worldTidesUpstreamFixture([]map[string]any{
					{"dt": int64(1), "height": 1.0, "type": "High"},
				})
				m["status"] = 400
				return mustMarshalJSON(t, m)
			},
			wantErr: errIncomingResponseStatus,
		},
		{
			name: "missing status",
			payload: func(t *testing.T) []byte {
				m := worldTidesUpstreamFixture([]map[string]any{
					{"dt": int64(1), "height": 1.0, "type": "High"},
				})
				delete(m, "status")
				return mustMarshalJSON(t, m)
			},
			wantErr: errIncomingResponseStatus,
		},
		{
			name: "empty copyright",
			payload: func(t *testing.T) []byte {
				m := worldTidesUpstreamFixture([]map[string]any{
					{"dt": int64(1), "height": 1.0, "type": "High"},
				})
				m["copyright"] = ""
				return mustMarshalJSON(t, m)
			},
			wantErr: errIncomingResponseCopyright,
		},
		{
			name: "missing copyright",
			payload: func(t *testing.T) []byte {
				m := worldTidesUpstreamFixture([]map[string]any{
					{"dt": int64(1), "height": 1.0, "type": "High"},
				})
				delete(m, "copyright")
				return mustMarshalJSON(t, m)
			},
			wantErr: errIncomingResponseCopyright,
		},
		{
			name: "responseDatum not CD",
			payload: func(t *testing.T) []byte {
				m := worldTidesUpstreamFixture([]map[string]any{
					{"dt": int64(1), "height": 1.0, "type": "High"},
				})
				m["responseDatum"] = "MSL"
				return mustMarshalJSON(t, m)
			},
			wantErr: errIncomingResponseDatum,
		},
		{
			name: "missing extremes key",
			payload: func(t *testing.T) []byte {
				m := map[string]any{
					"status":        200,
					"copyright":     "c",
					"responseDatum": "CD",
				}
				return mustMarshalJSON(t, m)
			},
			wantErr: errIncomingResponseExtremes,
		},
		{
			name: "extremes out of order",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"dt": int64(200), "height": 2.0, "type": "High"},
					{"dt": int64(100), "height": 1.0, "type": "Low"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			wantErr: errIncomingResponseExtremesOrder,
		},
		{
			name: "equal consecutive times",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"dt": int64(100), "height": 1.0, "type": "Low"},
					{"dt": int64(100), "height": 2.0, "type": "High"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			wantErr: errIncomingResponseExtremesOrder,
		},
		{
			name: "missing extreme dt",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"height": 1.0, "type": "High"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			wantErr: errIncomingResponseExtremeDt,
		},
		{
			name: "invalid extreme type",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"dt": int64(1), "height": 1.0, "type": "Mid"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			wantErr: errIncomingResponseExtremeType,
		},
		{
			name: "height not finite uses missing pointer",
			payload: func(t *testing.T) []byte {
				ext := []map[string]any{
					{"dt": int64(1), "type": "High"},
				}
				return mustMarshalJSON(t, worldTidesUpstreamFixture(ext))
			},
			wantErr: errIncomingResponseExtremeHeight,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseIncomingResponse(tt.payload(t))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: got %v want %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseIncomingResponse_jsonErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func(*testing.T) []byte
	}{
		{
			name: "invalid syntax",
			payload: func(*testing.T) []byte {
				return []byte{'{'}
			},
		},
		{
			name: "trailing second value",
			payload: func(t *testing.T) []byte {
				first := mustMarshalJSON(t, worldTidesUpstreamFixture([]map[string]any{}))
				second := mustMarshalJSON(t, map[string]any{})
				return slices.Concat(first, second)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseIncomingResponse(tt.payload(t))
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, errIncomingResponseStatus) || errors.Is(err, errIncomingResponseCopyright) ||
				errors.Is(err, errIncomingResponseDatum) || errors.Is(err, errIncomingResponseExtremes) ||
				errors.Is(err, errIncomingResponseExtremeDt) || errors.Is(err, errIncomingResponseExtremeType) ||
				errors.Is(err, errIncomingResponseExtremeHeight) || errors.Is(err, errIncomingResponseExtremesOrder) {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestParseIncomingResponse_trailingSecondValueUsesSentinel(t *testing.T) {
	t.Parallel()
	valid := mustMarshalJSON(t, worldTidesUpstreamFixture([]map[string]any{}))
	payload := slices.Concat(valid, []byte(`{}`))
	_, err := ParseIncomingResponse(payload)
	if !errors.Is(err, errIncomingResponseTrailingJSON) {
		t.Fatalf("error: got %v want %v", err, errIncomingResponseTrailingJSON)
	}
}

func TestParseIncomingResponse_emptyBody(t *testing.T) {
	t.Parallel()
	_, err := ParseIncomingResponse([]byte{})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %T %v", err, err)
	}
}
