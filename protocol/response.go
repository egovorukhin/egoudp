package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type Response struct {
	Id          string
	StatusCode  StatusCode
	Event       int
	ContentType string
	Data        Runes
}

type Runes []rune

func ToRunes(s string) Runes {
	return []rune(s)
}

func (r Runes) ToByte() []byte {
	return []byte(r.String())
}

func (r Runes) String() string {
	return string(r)
}

type IResponse interface {
	GetID() string
	SetData(code StatusCode, data Runes) *Response
	SetContentType(s string) *Response
	Marshal() []byte
	Unmarshal(b []byte) error
}

func NewResponse(req *Request, event int) IResponse {
	resp := &Response{
		Event: event,
	}
	if req != nil {
		resp.Id = req.Id
		resp.ContentType = req.ContentType
	}
	return resp
}

func (r *Response) GetID() string {
	return r.Id
}

func (r *Response) SetContentType(s string) *Response {
	r.ContentType = s
	return r
}

func (r *Response) SetData(code StatusCode, data Runes) *Response {
	r.StatusCode = code
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
	r.Event, _ = strconv.Atoi(event)
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
	r.Data = []rune(data)

	return
}

func (r *Response) String() string {
	data := "null"
	if r.Data != nil {
		data = fmt.Sprintf("%v", r.Data)
	}
	return fmt.Sprintf("id: %s, status_code: %s(%d), event: %s(%d), content_type: %s, data: %s",
		r.Id, r.StatusCode.String(), r.StatusCode, EventToString(Events(r.Event)), r.Event, r.ContentType, data)
}
