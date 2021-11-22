package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/vv/api"
)

func TestNeighborsHandlerGET(t *testing.T) {
	for label, tt := range map[string][]struct {
		label   string
		mpd     mpdNeighborsFunc
		err     error
		want    string
		changed bool
	}{
		"ok": {{
			label: "empty",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return []map[string]string{}, nil
			}),
			want: "{}",
		}, {
			label: "some data",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return []map[string]string{
					{
						"neighbor": "smb://FOO",
						"name":     "FOO (Samba 4.1.11-Debian)",
					},
				}, nil
			}),
			want:    `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			changed: true,
		}, {
			label: "remove",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return []map[string]string{}, nil
			}),
			want:    "{}",
			changed: true,
		}},
		"error/network": {{
			label: "prepare data",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return []map[string]string{
					{
						"neighbor": "smb://FOO",
						"name":     "FOO (Samba 4.1.11-Debian)",
					},
				}, nil
			}),
			want:    `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			changed: true,
		}, {
			label: "error",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return nil, errTest
			}),
			err:  errTest,
			want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
		}},
		"error/mpd": {{
			label: "prepare data",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return []map[string]string{
					{
						"neighbor": "smb://FOO",
						"name":     "FOO (Samba 4.1.11-Debian)",
					},
				}, nil
			}),
			want:    `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			changed: true,
		}, {
			label: "unknown command",
			mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
				return nil, &mpd.CommandError{ID: 5, Index: 0, Command: "listneighbors", Message: "unknown command \"listneighbors\""}
			}),
			want:    "{}",
			changed: true,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			f := make(mpdNeighborsFuncs, 1)
			h, err := api.NewNeighborsHandler(f)
			if err != nil {
				t.Fatalf("failed to init Neighbors: %v", err)
			}
			for i := range tt {
				t.Run(tt[i].label, func(t *testing.T) {
					f[0] = tt[i].mpd
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("Update(ctx) = %v; want %v", err, tt[i].err)
					}
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if status, got := w.Result().StatusCode, w.Body.String(); status != http.StatusOK || got != tt[i].want {
						t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, http.StatusOK, tt[i].want)
					}
					if changed := recieveMsg(h.Changed()); changed != tt[i].changed {
						t.Errorf("changed = %v; want %v", changed, tt[i].changed)
					}
				})
			}
		})
	}
}

type mpdNeighborsFunc func(context.Context) ([]map[string]string, error)

func (m mpdNeighborsFunc) ListNeighbors(ctx context.Context) ([]map[string]string, error) {
	return m(ctx)
}

type mpdNeighborsFuncs []mpdNeighborsFunc

func (m mpdNeighborsFuncs) ListNeighbors(ctx context.Context) ([]map[string]string, error) {
	return m[len(m)-1](ctx)
}
