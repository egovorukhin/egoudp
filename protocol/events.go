package protocol

import "fmt"

type Events int

const (
	EventNone Events = iota
	EventConnected
	EventDisconnect
	EventCheckConnection
)

func ToEvent(s string) Events {
	var event Events
	switch s {
	case "1":
		event = EventConnected
		break
	case "2":
		event = EventDisconnect
		break
	case "3":
		event = EventCheckConnection
		break
	default:
		event = EventNone
	}
	return event
}

func EventToString(event Events) string {
	return event.String()
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
