package protocol

import "fmt"

type Request struct {
	Route       string
	Id          string
	Method      Methods
	ContentType string
	Data        []byte
}

type IRequest interface {
	SetData(contentType string, data []byte) *Request
}

func NewRequest(route string, method Methods) IRequest {
	return &Request{
		Route:  route,
		Method: method,
	}
}

func (r *Request) SetData(contentType string, data []byte) *Request {
	r.ContentType = contentType
	r.Data = data
	return r
}

func (r *Request) String() string {
	data := "null"
	if r.Data != nil {
		data = fmt.Sprintf("%v", r.Data)
	}
	return fmt.Sprintf("Id: %s, method: %s, type: %s, data: %s", r.Id, r.Method.String(), r.ContentType, data)
}
