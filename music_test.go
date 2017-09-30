package main

import (
	"encoding/json"
	"errors"
	"github.com/meiraka/gompd/mpd"
	"reflect"
	"testing"
	"time"
)

func initMock(dialError, newWatcherError error) (*mockMpc, *mpd.Watcher) {
	m := new(mockMpc)
	m.ListAllInfoRet1 = []mpd.Tags{}
	m.PlaylistInfoRet1 = []mpd.Tags{}
	m.StatusRet1 = mpd.Attrs{}
	m.StatsRet1 = mpd.Attrs{}
	m.ReadCommentsRet1 = mpd.Attrs{}
	m.CurrentSongRet1 = mpd.Tags{}
	m.ListOutputsRet1 = []mpd.Attrs{}
	musicMpdDial = func(n, a, s string) (mpdClient, error) {
		m.DialCalled++
		return m, dialError
	}
	w := new(mpd.Watcher)
	musicMpdNewWatcher = func(n, a, s string) (*mpd.Watcher, error) {
		w.Event = make(chan string)
		m.NewWatcherCalled++
		return w, newWatcherError
	}
	musicMpdWatcherClose = func(w mpd.Watcher) error {
		close(w.Event)
		w.Event = nil
		return nil
	}
	return m, w
}

func TestDial(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, err := Dial("tcp", "localhost:6600", "", "./")
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 1 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	p.Close()
	me := new(mockError)
	m, _ = initMock(me, nil)
	p, err = Dial("tcp", "localhost:6600", "", "./")
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 0 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	p.Close()

	m, _ = initMock(nil, me)
	p, err = Dial("tcp", "localhost:6600", "", "./")
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 1 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	if m.CloseCalled != 1 {
		t.Errorf("mpd.Client.Close was not called: %d", m.CloseCalled)
	}
	p.Close()
}

func TestMusicWatch(t *testing.T) {
	_, w := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	p.watcherResponse = make(chan error, 10)
	defer p.Close()
	testsets := []struct {
		event     string
		responses int
	}{
		{event: "database", responses: 2},
		{event: "playlist", responses: 1},
		{event: "player", responses: 3},
		{event: "mixer", responses: 2},
		{event: "options", responses: 2},
		{event: "update", responses: 1},
		{event: "output", responses: 1},
	}
	for _, tt := range testsets {
		w.Event <- tt.event
		for i := 0; i < tt.responses; i++ {
			select {
			case err := <-p.watcherResponse:
				if err != nil {
					t.Errorf("unexpected error for %s: %s", tt.event, err.Error())
				}
			case <-time.After(10 * time.Millisecond):
				t.Errorf("timeout: no response for %s", tt.event)
			}
		}
	}
}

func TestMusicPlay(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	m.PlayRet1 = new(mockError)
	err := p.Play()
	if m.PlayCalled != 1 {
		t.Errorf("Client.Play does not Called")
	}
	if err != m.PlayRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PlayArg1 != -1 {
		t.Errorf("unexpected Client.Play Arguments: %d", m.PlayArg1)
	}

	m.PlayRet1 = nil
	err = p.Play()

	if m.PlayCalled != 2 {
		t.Errorf("Client.Play does not Called")
	}
	if err != m.PlayRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PlayArg1 != -1 {
		t.Errorf("unexpected Client.Play Arguments: %d", m.PlayArg1)
	}
	p.Close()
}

func TestMusicRescanLibrary(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	m.UpdateRet2 = nil
	err := p.RescanLibrary()
	if m.UpdateCalled != 1 {
		t.Errorf("Client.Update did not Called")
	}
	if m.UpdateArg1 != "" {
		t.Errorf("unexpected argument 1: %s", m.UpdateArg1)
	}
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestMusicPause(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	m.PauseRet1 = new(mockError)
	err := p.Pause()
	if m.PauseCalled != 1 {
		t.Errorf("Client.Pause does not Called")
	}
	if err != m.PauseRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PauseArg1 != true {
		t.Errorf("unexpected Client.Pause Arguments: %t", m.PauseArg1)
	}

	m.PauseRet1 = nil
	err = p.Pause()

	if m.PauseCalled != 2 {
		t.Errorf("Client.Pause does not Called")
	}
	if err != m.PauseRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PauseArg1 != true {
		t.Errorf("unexpected Client.Pause Arguments: %t", m.PauseArg1)
	}
}

func TestMusicNext(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	m.NextRet1 = new(mockError)
	err := p.Next()
	if m.NextCalled != 1 {
		t.Errorf("Client.Next does not Called")
	}
	if err != m.NextRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	m.NextRet1 = nil
	err = p.Next()
	if m.NextCalled != 2 {
		t.Errorf("Client.Next does not Called")
	}
	if err != m.NextRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestMusicPrevious(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	m.PreviousRet1 = new(mockError)
	err := p.Prev()
	if m.PreviousCalled != 1 {
		t.Errorf("Client.Previous does not Called")
	}
	if err != m.PreviousRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	m.PreviousRet1 = nil
	err = p.Prev()
	if m.PreviousCalled != 2 {
		t.Errorf("Client.Previous does not Called")
	}
	if err != m.PreviousRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestMusicSetVolume(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	err := p.Volume(1)
	if m.SetVolumeCalled != 1 {
		t.Errorf("Client.SetVolume does not Called")
	}
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestMusicRepeat(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	err := p.Repeat(true)
	if m.RepeatCalled != 1 {
		t.Errorf("Client.Repeat does not Called")
	}
	if m.RepeatArg1 != true {
		t.Errorf("unexpected argument: %t", m.RepeatArg1)
	}
	if err != m.RepeatRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestMusicRandom(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	err := p.Random(true)
	if m.RandomCalled != 1 {
		t.Errorf("Client.Random does not Called")
	}
	if m.RandomArg1 != true {
		t.Errorf("unexpected argument: %t", m.RandomArg1)
	}
	if err != m.RandomRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestMusicPlaylist(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	e := make(chan string, 1)
	p.Subscribe(e)
	defer p.Unsubscribe(e)
	if event := <-e; event != "stats" {
		t.Errorf("unexpected event. expect: stats, actual: %s", event)
	}
	m.PlaylistInfoCalled = 0
	m.PlaylistInfoRet1 = []mpd.Tags{{"foo": {"bar"}}}
	m.PlaylistInfoRet2 = nil
	expect := MakeSongs(m.PlaylistInfoRet1, p.musicDirectory, "cover.*", p.coverCache)
	p.updatePlaylist(m)
	// mpd.Client.PlaylistInfo was Called
	if m.PlaylistInfoCalled != 1 {
		t.Errorf("Client.PlaylistInfo does not Called")
	}
	if m.PlaylistInfoArg1 != -1 || m.PlaylistInfoArg2 != -1 {
		t.Errorf("unexpected Client.PlaylistInfo Arguments: %d %d", m.PlaylistInfoArg1, m.PlaylistInfoArg2)
	}
	// Music.Playlist returns mpd.Client.PlaylistInfo result
	playlist, _ := p.Playlist()
	if !reflect.DeepEqual(expect, playlist) {
		t.Errorf("unexpected get playlist")
	}
	if event := <-e; event != "playlist" {
		t.Errorf("unexpected event. expect: playlist, actual: %s", event)
	}
}

func TestMusicStats(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	e := make(chan string, 1)
	p.Subscribe(e)
	defer p.Unsubscribe(e)
	if event := <-e; event != "stats" {
		t.Errorf("unexpected event. expect: stats, actual: %s", event)
	}
	var testsets = []struct {
		desc   string
		ret1   mpd.Attrs
		ret2   error
		expect mpd.Attrs
		notify bool
	}{
		{
			desc: "no cache, error",
			ret1: nil, ret2: errors.New("hoge"),
			expect: mpd.Attrs{"subscribers": "1"},
			notify: false,
		},
		{
			desc: "no error",
			ret1: mpd.Attrs{"foo": "bar"}, ret2: nil,
			expect: mpd.Attrs{"foo": "bar", "subscribers": "1"},
			notify: true,
		},
		{
			desc: "use cache, error",
			ret1: nil, ret2: errors.New("hoge"),
			expect: mpd.Attrs{"foo": "bar", "subscribers": "1"},
			notify: false,
		},
	}
	for _, tt := range testsets {
		m.StatsRet1 = tt.ret1
		m.StatsRet2 = tt.ret2
		m.StatsCalled = 0
		err := p.updateStats(m)
		if err != tt.ret2 {
			t.Errorf("[%s] unexpected error: %s", tt.desc, err.Error())
		}
		actual, _ := p.Stats()
		if m.StatsCalled != 1 {
			t.Errorf("[%s] mpd.Client.Stats does not called", tt.desc)
		}
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("[%s] unexpected get stats", tt.desc)
		}
		if tt.notify {
			if event := <-e; event != "stats" {
				t.Errorf("[%s] unexpected stats event. expect: stats, actual: %s", tt.desc, event)
			}
		}

	}
}

func TestMusicLibrary(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	e := make(chan string, 1)
	p.Subscribe(e)
	defer p.Unsubscribe(e)
	if event := <-e; event != "stats" {
		t.Errorf("unexpected event. expect: stats, actual: %s", event)
	}
	m.ListAllInfoCalled = 0
	m.ListAllInfoRet1 = []mpd.Tags{{"foo": {"bar"}}}
	m.ListAllInfoRet2 = nil
	expect := MakeSongs(m.ListAllInfoRet1, p.musicDirectory, "cover.*", p.coverCache)
	p.updateLibrary(m)
	// mpd.Client.ListAllInfo was Called
	if m.ListAllInfoCalled != 1 {
		t.Errorf("Client.ListAllInfo does not Called")
	}
	if m.ListAllInfoArg1 != "/" {
		t.Errorf("unexpected Client.ListAllInfo Arguments: %s", m.ListAllInfoArg1)
	}
	// Music.Library returns mpd.Client.ListAllInfo result
	library, _ := p.Library()
	if !reflect.DeepEqual(expect, library) {
		t.Errorf("unexpected get library")
	}
	if event := <-e; event != "library" {
		t.Errorf("unexpected event. expect: library, actual: %s", event)
	}
}

func TestMusicCurrent(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	e := make(chan string, 1)
	p.Subscribe(e)
	defer p.Unsubscribe(e)
	if event := <-e; event != "stats" {
		t.Errorf("unexpected event. expect: stats, actual: %s", event)
	}
	m.CurrentSongCalled = 0
	m.StatusCalled = 0
	errret := new(mockError)
	candidates := []struct {
		CurrentSongRet1   mpd.Tags
		CurrentSongRet2   error
		CurrentSongCalled int
		currentRet        Song
		StatusRet1        mpd.Attrs
		StatusRet2        error
		StatusCalled      int
		StatusRet         Status
	}{
		// dont update if mpd.CurrentSong returns error
		{
			mpd.Tags{}, errret, 1,
			Song{},
			mpd.Attrs{}, errret, 1,
			MakeStatus(mpd.Attrs{}),
		},
		// update current/status/comments
		{
			mpd.Tags{"file": {"p"}}, nil, 2,
			MakeSong(mpd.Tags{"file": {"p"}}, p.musicDirectory, "cover.*", p.coverCache),
			mpd.Attrs{}, nil, 2,
			MakeStatus(mpd.Attrs{}),
		},
	}
	for _, c := range candidates {
		m.CurrentSongRet1 = c.CurrentSongRet1
		m.CurrentSongRet2 = c.CurrentSongRet2
		m.StatusRet1 = c.StatusRet1
		m.StatusRet2 = c.StatusRet2
		p.updateCurrentSong(m)
		if c.CurrentSongRet2 == nil {
			if event := <-e; event != "current" {
				t.Errorf("unexpected event. expect: current, actual: %s", event)
			}
		}
		p.updateStatus(m)
		if c.StatusRet2 == nil {
			if event := <-e; event != "status" {
				t.Errorf("unexpected event. expect: status, actual: %s", event)
			}
		}
		if m.CurrentSongCalled != c.CurrentSongCalled {
			t.Errorf("unexpected function call")
		}
		current, _ := p.Current()
		if !reflect.DeepEqual(current, c.currentRet) {
			t.Errorf(
				"unexpected Music.Current()\nexpect: %s\nactual:   %s",
				c.currentRet,
				current,
			)
		}
		if m.StatusCalled != c.StatusCalled {
			t.Errorf("unexpected function call")
		}
		status, _ := p.Status()
		if !reflect.DeepEqual(status, c.StatusRet) {
			sj, _ := json.Marshal(status)
			ej, _ := json.Marshal(c.StatusRet)
			t.Errorf(
				"unexpected Music.Status()\nexpect: %s\nactual:   %s",
				ej, sj,
			)
		}
	}
}

func TestMusicOutputs(t *testing.T) {
	m, _ := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	e := make(chan string, 1)
	p.Subscribe(e)
	defer p.Unsubscribe(e)
	if event := <-e; event != "stats" {
		t.Errorf("unexpected event. expect: stats, actual: %s", event)
	}
	m.ListOutputsCalled = 0
	m.ListOutputsRet1 = []mpd.Attrs{{"foo": "bar"}}
	m.ListOutputsRet2 = nil
	p.updateOutputs(m)
	// mpd.Client.ListOutputs was Called
	if m.ListOutputsCalled != 1 {
		t.Errorf("Client.ListOutputs does not Called")
	}
	// Music.Library returns mpd.Client.ListOutputs result
	outputs, _ := p.Outputs()
	if !reflect.DeepEqual(m.ListOutputsRet1, outputs) {
		t.Errorf("unexpected get outputs")
	}
	if event := <-e; event != "outputs" {
		t.Errorf("unexpected event. expect: outputs, actual: %s", event)
	}
}

type mockMpc struct {
	DialCalled             int
	NewWatcherCalled       int
	PlayCalled             int
	PlayArg1               int
	PlayRet1               error
	PauseCalled            int
	PauseArg1              bool
	PauseRet1              error
	NextCalled             int
	NextRet1               error
	PreviousCalled         int
	PreviousRet1           error
	CloseCalled            int
	CloseRet1              error
	SetVolumeCalled        int
	SetVolumeArg1          int
	SetVolumeRet1          error
	RepeatCalled           int
	RepeatArg1             bool
	RepeatRet1             error
	RandomCalled           int
	RandomArg1             bool
	RandomRet1             error
	PlaylistInfoCalled     int
	PlaylistInfoArg1       int
	PlaylistInfoArg2       int
	PlaylistInfoRet1       []mpd.Tags
	PlaylistInfoRet2       error
	ListAllInfoCalled      int
	ListAllInfoArg1        string
	ListAllInfoRet1        []mpd.Tags
	ListAllInfoRet2        error
	ReadCommentsCalled     int
	ReadCommentsArg1       string
	ReadCommentsRet1       mpd.Attrs
	ReadCommentsRet2       error
	CurrentSongCalled      int
	CurrentSongRet1        mpd.Tags
	CurrentSongRet2        error
	StatusCalled           int
	StatusRet1             mpd.Attrs
	StatusRet2             error
	StatsCalled            int
	StatsRet1              mpd.Attrs
	StatsRet2              error
	PingCalled             int
	PingRet1               error
	ListOutputsCalled      int
	ListOutputsRet1        []mpd.Attrs
	ListOutputsRet2        error
	DisableOutputCalled    int
	DisableOutputArg1      int
	DisableOutputRet1      error
	EnableOutputCalled     int
	EnableOutputArg1       int
	EnableOutputRet1       error
	UpdateCalled           int
	UpdateArg1             string
	UpdateRet1             int
	UpdateRet2             error
	begincommandlistCalled int
}

func (p *mockMpc) Play(PlayArg1 int) error {
	p.PlayCalled++
	p.PlayArg1 = PlayArg1
	return p.PlayRet1
}
func (p *mockMpc) Pause(PauseArg1 bool) error {
	p.PauseCalled++
	p.PauseArg1 = PauseArg1
	return p.PauseRet1
}
func (p *mockMpc) Next() error {
	p.NextCalled++
	return p.NextRet1
}
func (p *mockMpc) Previous() error {
	p.PreviousCalled++
	return p.PreviousRet1
}
func (p *mockMpc) Close() error {
	p.CloseCalled++
	return p.CloseRet1
}
func (p *mockMpc) SetVolume(i int) error {
	p.SetVolumeCalled++
	p.SetVolumeArg1 = i
	return p.SetVolumeRet1
}
func (p *mockMpc) Repeat(b bool) error {
	p.RepeatCalled++
	p.RepeatArg1 = b
	return p.RepeatRet1
}
func (p *mockMpc) Random(b bool) error {
	p.RandomCalled++
	p.RandomArg1 = b
	return p.RandomRet1
}
func (p *mockMpc) Ping() error {
	p.PingCalled++
	return p.PingRet1
}
func (p *mockMpc) CurrentSongTags() (mpd.Tags, error) {
	p.CurrentSongCalled++
	return p.CurrentSongRet1, p.CurrentSongRet2
}
func (p *mockMpc) Status() (mpd.Attrs, error) {
	p.StatusCalled++
	return p.StatusRet1, p.StatusRet2
}
func (p *mockMpc) Stats() (mpd.Attrs, error) {
	p.StatsCalled++
	return p.StatsRet1, p.StatsRet2
}
func (p *mockMpc) ReadComments(ReadCommentsArg1 string) (mpd.Attrs, error) {
	p.ReadCommentsCalled++
	p.ReadCommentsArg1 = ReadCommentsArg1
	return p.ReadCommentsRet1, p.ReadCommentsRet2
}
func (p *mockMpc) PlaylistInfoTags(PlaylistInfoArg1, PlaylistInfoArg2 int) ([]mpd.Tags, error) {
	p.PlaylistInfoCalled++
	p.PlaylistInfoArg1 = PlaylistInfoArg1
	p.PlaylistInfoArg2 = PlaylistInfoArg2
	return p.PlaylistInfoRet1, p.PlaylistInfoRet2
}
func (p *mockMpc) ListAllInfoTags(ListAllInfoArg1 string) ([]mpd.Tags, error) {
	p.ListAllInfoCalled++
	p.ListAllInfoArg1 = ListAllInfoArg1
	return p.ListAllInfoRet1, p.ListAllInfoRet2
}

func (p *mockMpc) ListOutputs() ([]mpd.Attrs, error) {
	p.ListOutputsCalled++
	return p.ListOutputsRet1, p.ListOutputsRet2
}

func (p *mockMpc) DisableOutput(arg1 int) error {
	p.DisableOutputCalled++
	p.DisableOutputArg1 = arg1
	return p.DisableOutputRet1
}

func (p *mockMpc) EnableOutput(arg1 int) error {
	p.EnableOutputCalled++
	p.EnableOutputArg1 = arg1
	return p.EnableOutputRet1
}

func (p *mockMpc) BeginCommandList() *mpd.CommandList {
	p.begincommandlistCalled++
	return nil
}

func (p *mockMpc) Update(arg1 string) (int, error) {
	p.UpdateCalled++
	p.UpdateArg1 = arg1
	return p.UpdateRet1, p.UpdateRet2
}

type mockError struct{}

func (m *mockError) Error() string { return "err" }