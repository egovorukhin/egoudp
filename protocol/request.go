package protocol

import (
	"fmt"
)

type Request struct {
	Path        string
	Method      Methods
	Id          string
	ContentType string
	Data        []byte
}

type IRequest interface {
	SetData(contentType string, data []byte) *Request
}

func NewRequest(path string, method Methods) *Request {
	return &Request{
		Path:   path,
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
	return fmt.Sprintf("Id: %s, path: %s, method: %s, type: %s, data: %s", r.Id, r.Path, r.Method.String(), r.ContentType, data)
}
