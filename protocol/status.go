package protocol

import "fmt"

type StatusCode int

const (
	StatusCodeOK StatusCode = iota
	StatusCodeError
)

func ToStatusCode(s string) StatusCode {
	var code StatusCode
	switch s {
	case "0":
		code = StatusCodeOK
		break
	default:
		code = StatusCodeError
	}
	return code
}

func (sc StatusCode) String() string {
	s := "StatusCodeError"
	switch sc {
	case StatusCodeOK:
		s = "StatusCodeOK"
		break
	}
	return fmt.Sprintf("%s(%d)", s, sc)
}
