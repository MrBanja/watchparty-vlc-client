package web

type Status string

const (
	Play  Status = "play"
	Pause Status = "pause"
)

type Message struct {
	Time   int
	Status Status
}
