package server

/*
type Message struct {
	Id         string     `json:"id"`
	Command    Commands   `json:"command"`
	Method     Methods    `json:"method"`
	StatusCode StatusCode `json:"status_code"`
	Data       *Data      `json:"data,omitempty"`
}

func NewSendMessage(command Commands, method Methods) *Message {
	return &Message{
		Command: command,
		Method:  method,
	}
}

func (s *Message) Json(data interface{}) (*Message, error) {
	if data == nil {
		return s, nil
	}
	s.Data = &Data{
		ContentType: Json,
	}
	return s.marshal(data)
}

func (s *Message) Xml(data interface{}) (*Message, error) {
	if data == nil {
		return s, nil
	}
	s.Data = &Data{
		ContentType: Xml,
	}
	return s.marshal(data)
}

func (s *Message) Text(data interface{}) (*Message, error) {
	if data == nil {
		return s, nil
	}
	s.Data = &Data{
		ContentType: Text,
	}
	return s.marshal(data)
}

func (s *Message) marshal(data interface{}) (*Message, error) {
	tmp, err := s.Data.ContentType.Marshal(data)
	if err != nil {
		return nil, err
	}
	s.Data.Body = string(tmp)
	return s, nil
}

func (m Message) Response(hostname string, statusCode StatusCode, data *Data) {
	m.Data = data
	m.StatusCode = statusCode
	err := SendByHostname(hostname, &m)
	if err != nil {
		egologger.New(m.Response, "server").Error(err.Error())
	}
}

func (m Message) OK(hostname string, data *Data) {
	m.Response(hostname, StatusCodeOK, data)
}

func (m Message) Error(hostname string, data *Data) {
	m.Response(hostname, StatusCodeError, data)
}*/
