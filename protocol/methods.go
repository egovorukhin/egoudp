package protocol

import "fmt"

type Methods int

const (
	MethodNone Methods = iota
	MethodGet
	MethodSet
)

func ToMethod(s string) Methods {
	var method Methods
	switch s {
	case "1":
		method = MethodGet
		break
	case "2":
		method = MethodSet
	default:
		method = MethodNone
	}
	return method
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
