package protocol

import "fmt"

type Methods int

const (
	MethodNone Methods = iota
	MethodGet
	MethodSet
)

func (m *Methods) unmarshal(s string) {
	var method Methods
	switch s {
	case "0":
		method = MethodGet
		break
	case "1": method = MethodSet
	default:
		method = MethodNone
	}
	m = &method
}

func (m Methods) String() string {
	s := "MethodNone"
	switch m {
	case MethodGet:
		s = "MethodGet"
		break
	case MethodSet:
		s = "MethodSet"
		break
	}
	return fmt.Sprintf("%s(%d)", s, m)
}
