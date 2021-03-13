package protocol

import "fmt"

type Header struct {
	Hostname string
	Login    string
	Domain   string
	Version  string
	Event    int
}

func (h Header) IsNil() bool {
	if h.Hostname == "" ||
		h.Login == "" ||
		h.Domain == "" ||
		h.Version == "" {
		return true
	}
	return false
}

func (h *Header) String() string {
	return fmt.Sprintf("hostname: %s, login: %s, domain: %s, version: %s, event: %s(%d)",
		h.Hostname, h.Login, h.Domain, h.Version, EventToString(Events(h.Event)), h.Event)
}
