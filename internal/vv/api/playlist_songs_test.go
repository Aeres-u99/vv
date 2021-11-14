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

func TestPlaylistSongsGet(t *testing.T) {
	songsHook, randValue := testSongsHook()
	for label, tt := range map[string][]struct {
		playlistInfo func(*testing.T) ([]map[string][]string, error)
		err          error
		want         string
		eventHook    []map[string][]string
		changed      bool
	}{
		`empty`: {{
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				return []map[string][]string{}, nil
			},
			want: `[]`,
		}},
		`exists`: {{
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:      fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			eventHook: []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed:   true,
		}},
		`error after exists`: {{
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:      fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			eventHook: []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed:   true,
		}, {
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				t.Helper()
				return nil, context.DeadlineExceeded
			},
			err:  context.DeadlineExceeded,
			want: fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdPlaylistSongsAPI{t: t}

			var eventHook func([]map[string][]string)
			h, err := api.NewPlaylistSongs(mpd, songsHook, func(s []map[string][]string) { eventHook(s) })
			if err != nil {
				t.Fatalf("api.NewPlaylistSongs() = %v, %v", h, err)
			}
			for i := range tt {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					called := false
					eventHook = func(got []map[string][]string) {
						t.Helper()
						called = true
						if !reflect.DeepEqual(got, tt[i].eventHook) {
							t.Errorf("call eventHook(%q); want eventHook(%q)", got, tt[i].eventHook)
						}
					}
					mpd.playlistInfo = tt[i].playlistInfo
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
					changed := false
					select {
					case <-h.Changed():
						changed = true
					default:
					}
					if changed != tt[i].changed {
						t.Errorf("changed = %v; want %v", changed, tt[i].changed)
					}
					if called != tt[i].changed {
						t.Errorf("eventHook called = %v; want %v", called, tt[i].changed)
					}
				})
			}
		})
	}

}

type mpdPlaylistSongsAPI struct {
	t            *testing.T
	playlistInfo func(*testing.T) ([]map[string][]string, error)
}

func (m *mpdPlaylistSongsAPI) PlaylistInfo(ctx context.Context) ([]map[string][]string, error) {
	m.t.Helper()
	if m.playlistInfo == nil {
		m.t.Fatal("no PlaylistInfo mock function")
	}
	return m.playlistInfo(m.t)
}
