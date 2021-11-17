package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestLibrarySongsGet(t *testing.T) {
	songsHook, randValue := testSongsHook()
	for label, tt := range map[string][]struct {
		listAllInfo func(*testing.T, string) ([]map[string][]string, error)
		err         error
		want        string
		cache       []map[string][]string
		changed     bool
	}{
		`empty`: {{
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("got mpd.ListAllInfo(..., %q); want mpd.ListAllInfo(..., %q)", path, "/")
				}
				return []map[string][]string{}, nil
			},
			want:    `[]`,
			cache:   []map[string][]string{},
			changed: true,
		}},
		`exists`: {{
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("got mpd.ListAllInfo(..., %q); want mpd.ListAllInfo(..., %q)", path, "/")
				}
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:    fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache:   []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed: true,
		}},
		`error after exists`: {{
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("got mpd.ListAllInfo(..., %q); want mpd.ListAllInfo(..., %q)", path, "/")
				}
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:    fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache:   []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed: true,
		}, {
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("got mpd.ListAllInfo(..., %q); want mpd.ListAllInfo(..., %q)", path, "/")
				}
				return nil, context.DeadlineExceeded
			},
			err:   context.DeadlineExceeded,
			want:  fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache: []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdLibrarySongs{t: t}
			h, err := api.NewLibrarySongsHandler(mpd, songsHook)
			if err != nil {
				t.Fatalf("api.NewLibrarySongs() = %v, %v", h, err)
			}
			for i := range tt {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					mpd.listAllInfo = tt[i].listAllInfo
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("handler.Update(context.TODO()) = %v; want %v", err, tt[i].err)
					}

					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if got := w.Body.String(); got != tt[i].want {
						t.Errorf("ServeHTTP response: got\n%s; want\n%s", got, tt[i].want)
					}
					if got, want := w.Result().StatusCode, http.StatusOK; got != want {
						t.Errorf("ServeHTTP response status: got %d; want %d", got, want)
					}
					if cache := h.Cache(); !reflect.DeepEqual(cache, tt[i].cache) {
						t.Errorf("got cache\n%v; want\n%v", cache, tt[i].cache)
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

func testSongsHook() (func(s []map[string][]string) []map[string][]string, string) {
	f, key := testSongHook()
	return func(s []map[string][]string) []map[string][]string {
		for i := range s {
			s[i] = f(s[i])
		}
		return s
	}, key
}

type mpdLibrarySongs struct {
	t           *testing.T
	listAllInfo func(*testing.T, string) ([]map[string][]string, error)
}

func (m *mpdLibrarySongs) ListAllInfo(ctx context.Context, s string) ([]map[string][]string, error) {
	m.t.Helper()
	if m.listAllInfo == nil {
		m.t.Fatal("no ListAllInfo mock function")
	}
	return m.listAllInfo(m.t, s)
}
