package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/vv/api"
)

func TestOutputsHandlerGet(t *testing.T) {
	proxy := map[string]string{"Ogg Stream": "localhost:8080/"}
	for label, tt := range map[string][]struct {
		outputs func() ([]*mpd.Output, error)
		err     error
		want    string
		changed bool
	}{
		"empty/ok/empty": {{
			outputs: func() ([]*mpd.Output, error) { return []*mpd.Output{}, nil },
			want:    `{}`,
		}, {
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:      "0",
					Name:    "My ALSA Device",
					Plugin:  "alsa",
					Enabled: true,
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true}}`,
			changed: true,
		}, {
			outputs: func() ([]*mpd.Output, error) { return []*mpd.Output{}, nil },
			want:    `{}`,
			changed: true,
		}},
		"ok/dop": {{
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"dop": "0"},
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}}}`,
			changed: true,
		}},
		"ok/allowed formats": {{
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"allowed_formats": "96000:16:* 192000:24:* dsd64:=dop *:dsd:"},
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd64:=dop","*:dsd:"]}}}`,
			changed: true,
		}},
		"ok/stream": {{
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"dop": "0"},
				}, {
					ID:      "1",
					Name:    "Ogg Stream",
					Plugin:  "http",
					Enabled: true,
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}},"1":{"name":"Ogg Stream","plugin":"http","enabled":true,"stream":"/api/music/outputs/stream?name=Ogg+Stream"}}`,
			changed: true,
		}},
		"error": {{
			outputs: func() ([]*mpd.Output, error) { return nil, context.DeadlineExceeded },
			err:     context.DeadlineExceeded,
			want:    `{}`,
		}, {
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"dop": "0"},
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}}}`,
			changed: true,
		}, {
			outputs: func() ([]*mpd.Output, error) { return nil, context.DeadlineExceeded },
			err:     context.DeadlineExceeded,
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}}}`,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdOutputs{t: t}
			h, err := api.NewOutputsHandler(mpd, proxy)
			if err != nil {
				t.Fatalf("api.NewOutputsHandler(mpd) = %v", err)
			}
			defer h.Close()
			for i := range tt {
				t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
					mpd.t = t
					mpd.outputs = tt[i].outputs
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("h.Update(context.TODO()) = %v; want %v", err, tt[i].err)
					}

					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if status, got := w.Result().StatusCode, w.Body.String(); status != http.StatusOK || got != tt[i].want {
						t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, http.StatusOK, tt[i].want)
					}
					changed := false
					select {
					case <-h.Changed():
						changed = true
					default:
					}
					if changed != tt[i].changed {
						t.Errorf("changed = %v; want %v", changed, tt[i].changed)
					}
				})
			}
		})
	}
}

type mpdOutputs struct {
	t             *testing.T
	enableOutput  func(*testing.T, string) error
	disableOutput func(*testing.T, string) error
	outputSet     func(*testing.T, string, string, string) error
	outputs       func() ([]*mpd.Output, error)
}

func (m *mpdOutputs) EnableOutput(ctx context.Context, a string) error {
	m.t.Helper()
	if m.enableOutput == nil {
		m.t.Fatal("no EnableOutput mock function")
	}
	return m.enableOutput(m.t, a)
}
func (m *mpdOutputs) DisableOutput(ctx context.Context, a string) error {
	m.t.Helper()
	if m.disableOutput == nil {
		m.t.Fatal("no DisableOutput mock function")
	}
	return m.disableOutput(m.t, a)
}
func (m *mpdOutputs) OutputSet(ctx context.Context, a string, b string, c string) error {
	m.t.Helper()
	if m.outputSet == nil {
		m.t.Fatal("no OutputSet mock function")
	}
	return m.outputSet(m.t, a, b, c)
}
func (m *mpdOutputs) Outputs(context.Context) ([]*mpd.Output, error) {
	m.t.Helper()
	if m.outputs == nil {
		m.t.Fatal("no Outputs mock function")
	}
	return m.outputs()
}
