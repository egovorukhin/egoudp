package server

import "github.com/egovorukhin/egologger"

type Response struct {
	Id         string     `json:"id"`
	Command    Commands   `json:"command"`
	StatusCode StatusCode `json:"status_code"`
	Data       *Data      `json:"data,omitempty"`
}

type StatusCode int

const (
	StatusCodeOK StatusCode = iota
	StatusCodeError
)

func NewResponse(id string, command Commands) *Response {
	return &Response{
		Id:      id,
		Command: command,
	}
}

func SetResponse(request Request) *Response {
	return NewResponse(request.Id, request.Command)
}

func (m Response) Send(hostname string, statusCode StatusCode, data *Data) {
	m.Data = data
	m.StatusCode = statusCode
	err := SendByHostname(hostname, &m)
	if err != nil {
		egologger.New(m.Send, "server").Error(err.Error())
	}
}

func (m Response) OK(hostname string, data *Data) {
	m.Send(hostname, StatusCodeOK, data)
}

func (m Response) Error(hostname string, data *Data) {
	m.Send(hostname, StatusCodeError, data)
}
