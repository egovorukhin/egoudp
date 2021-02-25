package protocol

type Response struct {
	id         string
	StatusCode StatusCode
	Type       string
	Data       []byte
}
/*
func NewResponse() *Response {
	return &Response{
		id:         id,
		StatusCode: code,
	}
}

func (m *Response) OK(hostname string) {
	m.Send(hostname, StatusCodeOK, m)
}

func (m *Response) Error(hostname string) {
	m.Send(hostname, StatusCodeError, m)
}*/
