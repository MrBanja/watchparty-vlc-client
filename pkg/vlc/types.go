package vlc

import (
	"errors"
	"sync"

	vlcctrl "github.com/CedArctic/go-vlc-ctrl"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Callback func(status vlcctrl.Status) error

type VLC struct {
	vlc    *vlcctrl.VLC
	logger *zap.Logger

	mu        sync.Mutex
	prevState State
	statusCh  chan Status
}

type By int

const (
	ByUnknown By = iota
	ByUser
	ByNet
	BySystem
)

func (b By) MarshalLogObject(o zapcore.ObjectEncoder) error {
	switch b {
	case ByUnknown:
		o.AddString("By", "Unknown")
	case ByUser:
		o.AddString("By", "User")
	case ByNet:
		o.AddString("By", "Net")
	case BySystem:
		o.AddString("By", "System")
	default:
		return errors.New("unknown By")
	}
	return nil
}

type Status struct {
	Time  uint   `json:"time"`
	State string `json:"state"`
	By    By     `json:"-"`
}

type State int

const (
	StateUnknown State = iota
	StateStopped
	StatePaused
	StatePlaying
)

func GetPlayingState(state string) State {
	switch state {
	case "stopped":
		return StateStopped
	case "paused":
		return StatePaused
	case "playing":
		return StatePlaying
	default:
		return StateUnknown
	}
}
