package protocol

import "fmt"

type Header struct {
	Hostname string `eup:"hostname"`
	Login    string `eup:"login"`
	Domain   string `eup:"domain"`
	Version  string `eup:"version"`
	Event    Events `eup:"event"`
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
