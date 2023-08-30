package vlc

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	vlcctrl "github.com/CedArctic/go-vlc-ctrl"
	"go.uber.org/zap"
)

func MustNew(ctx context.Context, logger *zap.Logger) *VLC {
	logger = logger.Named("VLC")
	myVLC, err := vlcctrl.NewVLC("127.0.0.1", 8080, "qwerty")
	if err != nil {
		logger.Panic("Error creating VLC client", zap.Error(err))
	}

	if err := myVLC.EmptyPlaylist(); err != nil {
		logger.Panic("Error clearing playlist", zap.Error(err))
	}
	logger.Info("Playlist cleared")
	vlc := &VLC{
		vlc:      &myVLC,
		logger:   logger,
		statusCh: make(chan Status, 10),
	}
	vlc.prevState = StateStopped
	go vlc.userStatusCheck(ctx)
	return vlc
}

func (v *VLC) EnforceLogger(logger *zap.Logger) {
	v.logger = logger.Named("VLC")
}

func ParseStatus(statusResponse string) (status Status, err error) {
	err = json.Unmarshal([]byte(statusResponse), &status)
	if err != nil {
		return
	}
	return status, nil
}

func (v *VLC) StatusCh() <-chan Status {
	return v.statusCh
}

func (v *VLC) Add(pathFile string) error {
	defer v.logger.Info("File added to playlist", zap.String("Path", pathFile))
	return v.vlc.Add(url.PathEscape("file:///" + pathFile))
}

func (v *VLC) PlayBy(by By) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.prevState == StatePlaying {
		defer v.logger.Info("Already Playing", zap.Object("Called", by))
		return nil
	}
	defer v.logger.Info("Playing by", zap.Object("Called", by))
	urlSegment := "/requests/status.json?command=pl_play"
	response, err := v.vlc.RequestMaker(urlSegment)
	if err != nil {
		return err
	}

	status, err := ParseStatus(response)
	if err != nil {
		return err
	}

	v.prevState = StatePlaying
	status.By = by
	v.statusCh <- status
	return nil
}

func (v *VLC) PauseBy(by By) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.prevState == StatePaused {
		defer v.logger.Info("Already Paused", zap.Object("Called", by))
		return nil
	}
	defer v.logger.Info("Paused", zap.Object("Called", by))
	urlSegment := "/requests/status.json?command=pl_pause"
	response, err := v.vlc.RequestMaker(urlSegment)
	if err != nil {
		return err
	}

	status, err := ParseStatus(response)
	if err != nil {
		return err
	}

	v.prevState = StatePaused
	status.By = by
	v.statusCh <- status
	return nil
}

func (v *VLC) Seek(time int) error {
	defer v.logger.Info("Seeking", zap.Int("To Time", time))
	return v.vlc.Seek(strconv.Itoa(time))
}

func (v *VLC) userStatusCheck(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				v.logger.Info("Context DONE received. Stopping status check")
				return
			case <-ticker.C:
				v.mu.Lock()
				s, _ := v.vlc.GetStatus()
				if GetPlayingState(s.State) != v.prevState {
					v.prevState = GetPlayingState(s.State)
					v.statusCh <- Status{
						Time:  s.Time,
						State: s.State,
						By:    ByUser,
					}
				}
				v.mu.Unlock()
			}
		}
	}()
}
