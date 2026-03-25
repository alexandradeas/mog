package module

type Status int

const (
	StatusIdle Status = iota
	StatusRunning
	StatusCompleted
	StatusErrored
)

var statusMap = map[Status]string{
	StatusIdle:      "Idle",
	StatusRunning:   "Running",
	StatusCompleted: "Completed",
	StatusErrored:   "Errored",
}
