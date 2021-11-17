package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestLibraryHandlerGET(t *testing.T) {
	for label, tt := range map[string][]struct {
		err          error
		want         string
		changed      bool
		updateStatus *bool
	}{
		"default": {{
			want: `{"updating":false}`,
		}},
		"nochanged": {{
			updateStatus: boolptr(false),
			want:         `{"updating":false}`,
		}},
		"updating": {{
			want: `{"updating":false}`,
		}, {
			updateStatus: boolptr(true),
			want:         `{"updating":true}`,
			changed:      true,
		}, {
			updateStatus: boolptr(false),
			want:         `{"updating":false}`,
			changed:      true,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdLibrary{t: t}
			h, err := api.NewLibraryHandler(mpd)
			if err != nil {
				t.Fatalf("api.NewLibraryHandler(mpd) = %v", err)
			}
			defer h.Close()
			for i := range tt {
				t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
					mpd.t = t
					if tt[i].updateStatus != nil {
						if err := h.UpdateStatus(*tt[i].updateStatus); !errors.Is(err, tt[i].err) {
							t.Errorf("handler.UpdateStatus(%v) = %v; want %v", *tt[i].updateStatus, err, tt[i].err)
						}
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

func TestLibraryHandlerPOST(t *testing.T) {
	for label, tt := range map[string]struct {
		body       string
		wantStatus int
		want       string
		update     func(*testing.T, string) (map[string]string, error)
	}{
		"updating/true": {
			body:       `{"updating":true}`,
			want:       `{"updating":false}`,
			wantStatus: http.StatusAccepted,
			update: func(t *testing.T, a string) (map[string]string, error) {
				t.Helper()
				if want := ""; a != want {
					t.Errorf("call (*mpd.MPD).Update(ctx, %q); want (*mpd.MPD).Update(ctx, %q)", a, want)
				}
				return map[string]string{"updating": "1"}, nil
			},
		},
		"updating/true/error": {
			body:       `{"updating":true}`,
			want:       fmt.Sprintf(`{"error":"%s"}`, context.DeadlineExceeded.Error()),
			wantStatus: http.StatusInternalServerError,
			update: func(t *testing.T, a string) (map[string]string, error) {
				t.Helper()
				if want := ""; a != want {
					t.Errorf("call (*mpd.MPD).Update(ctx, %q); want (*mpd.MPD).Update(ctx, %q)", a, want)
				}
				return nil, context.DeadlineExceeded
			},
		},
		"updating/false": {
			body:       `{"updating":false}`,
			want:       `{"error":"requires updating=true"}`,
			wantStatus: http.StatusBadRequest,
		},
		"invalid json": {
			body:       `not json`,
			want:       `{"error":"invalid character 'o' in literal null (expecting 'u')"}`,
			wantStatus: http.StatusBadRequest,
		},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdLibrary{
				t:      t,
				update: tt.update,
			}
			h, err := api.NewLibraryHandler(mpd)
			if err != nil {
				t.Fatalf("api.NewLibraryHandler(mpd) = %v, %v", h, err)
			}
			defer h.Close()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if status, got := w.Result().StatusCode, w.Body.String(); status != tt.wantStatus || got != tt.want {
				t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, tt.wantStatus, tt.want)
			}
		})
	}
}

type mpdLibrary struct {
	t      *testing.T
	update func(*testing.T, string) (map[string]string, error)
}

func (m *mpdLibrary) Update(ctx context.Context, a string) (map[string]string, error) {
	m.t.Helper()
	if m.update == nil {
		m.t.Fatal("no Update mock function")
	}
	return m.update(m.t, a)
}
