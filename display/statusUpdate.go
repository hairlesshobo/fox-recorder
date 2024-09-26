package display

type StatusUpdate struct {
	Time   int64
	Status string
}

type SignalLevel struct {
	Instant int
	Max     int
	Peak    int
}
