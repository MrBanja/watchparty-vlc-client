package ui

type Progress interface {
	Set(int)
	Get() int
}
