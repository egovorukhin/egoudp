package protocol

import "fmt"

type Header struct {
	Hostname string
	Login    string
	Domain   string
	Version  string
	Event    Events
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
	return fmt.Sprintf("hostname: %s, login: %s, domain: %s, version: %s, event: %s",
		h.Hostname, h.Login, h.Domain, h.Version, h.Event.String())
}
