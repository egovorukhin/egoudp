package protocol

import "fmt"

type Events int

const (
	EventNone Events = iota
	EventConnected
	EventDisconnect
)

func (e *Events) unmarshal(s string) {
	var event Events
	switch s {
	case "0":
		event = EventConnected
		break
	case "1":
		event = EventDisconnect
		break
	default:
		event = EventNone
	}
	e = &event
}

func (e Events) String() string {
	s := "EventNone"
	switch e {
	case EventConnected:
		s = "EventConnected"
		break
	case EventDisconnect:
		s = "EventDisconnect"
		break
	}
	return fmt.Sprintf("%s(%d)", s, e)
}
