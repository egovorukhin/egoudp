package protocol

import (
	"bytes"
	"errors"
	"fmt"
)

type Response struct {
	Id          string
	StatusCode  StatusCode
	Event       Events
	ContentType string
	Data        []byte
}

type IResponse interface {
	OK(data []byte) *Response
	Error(data []byte) *Response
	SetData(data []byte) *Response
	SetContentType(s string) *Response
	Marshal() []byte
	Unmarshal(b []byte) error
}

func NewResponse(req *Request, event Events) *Response {
	resp := &Response{
		Event: event,
	}
	if req != nil {
		resp.Id = req.Id
		resp.ContentType = req.ContentType
	}
	return resp
}

func (r *Response) OK(data []byte) *Response {
	r.StatusCode = StatusCodeOK
	r.SetData(data)
	return r
}

func (r *Response) Error(data []byte) *Response {
	r.StatusCode = StatusCodeError
	r.SetData(data)
	return r
}

func (r *Response) SetContentType(s string) *Response {
	r.ContentType = s
	return r
}

func (r *Response) SetData(data []byte) *Response {
	r.Data = data
	return r
}

func (r *Response) Marshal() (b []byte) {
	buf := bytes.NewBuffer(b)
	buf.Write([]byte(string(startChar)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(r.Id), r.Id)))
	buf.Write([]byte(fmt.Sprintf("1:%d", r.StatusCode)))
	buf.Write([]byte(fmt.Sprintf("1:%d", r.Event)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(r.ContentType), r.ContentType)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(r.Data), r.Data)))
	buf.Write([]byte(string(endChar)))

	return buf.Bytes()
}

func (r *Response) Unmarshal(b []byte) (err error) {

	if b[0] != startChar {
		return errors.New(fmt.Sprintf("Первый символ должен быть - %v", startChar))
	}
	if b[len(b)-1] != endChar {
		return errors.New(fmt.Sprintf("Последний символ должен быть - %v", endChar))
	}

	//1. Id
	r.Id, b, err = findField(b[1:])
	if err != nil {
		return
	}
	//2. status-code
	code, b, err := findField(b)
	if err != nil {
		return
	}
	r.StatusCode = ToStatusCode(code)
	//3. event
	event, b, err := findField(b)
	if err != nil {
		return
	}
	r.Event = ToEvent(event)
	//4. content-type
	r.ContentType, b, err = findField(b)
	if err != nil {
		return
	}
	//5. data
	data, b, err := findField(b)
	if err != nil {
		return
	}
	r.Data = []byte(data)

	return
}

func (r *Response) String() string {
	data := "null"
	if r.Data != nil {
		data = fmt.Sprintf("%v", r.Data)
	}
	return fmt.Sprintf("Id: %s, status_code: %d, content_type: %s, data: %s", r.Id, r.StatusCode, r.ContentType, data)
}
