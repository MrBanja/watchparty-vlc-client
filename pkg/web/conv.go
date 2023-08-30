package web

import protocol "github.com/mrbanja/watchparty-proto/gen-go"

func dto2dao_State(state protocol.StateType) Status {
	switch state {
	case protocol.StateType_PLAY:
		return Play
	case protocol.StateType_PAUSE:
		return Pause
	}
	return Pause
}

func dao2dto_State(state Status) protocol.StateType {
	switch state {
	case Play:
		return protocol.StateType_PLAY
	case Pause:
		return protocol.StateType_PAUSE
	}
	return protocol.StateType_PAUSE
}
