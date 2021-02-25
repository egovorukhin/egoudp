package server

/*
type Receiver struct {
	Header protocol.Header `json:"header"`
	//Body   []Message `json:"body,omitempty"`
	Body *Request `json:"body,omitempty"`
}

type IReceiver interface {
	Parse(Request)
}

func NewReceiver() IReceiver {
	return &Receiver{
		Header: protocol.Header{},
		Body:   nil,
	}
}

type HandleParse func()
/*
type ReceiveMessage struct {
	Header protocol.Header `json:"header"`
	//Body   []Message `json:"body,omitempty"`
	Body *Request `json:"body,omitempty"`
}

func (r *Receiver) Parse(request Request) (err error) {

	//Инициализируем Request
	//request := message.Body
	//Инициализируем Response
	response := SetResponse(request)

	var buf []byte
	if request.Data != nil {
		fmt.Println(request.Data.Body)
		buf = []byte(request.Data.Body.(string))
	}

	switch request.Command {
	//Отправляем команду о подключении клиенту
	case CommandConnected:
		response.OK(r.Header.Hostname, nil)
		return
	//Команда на отключение клиента
	case CommandDisconnected:
		response.OK(r.Header.Hostname, nil)
		r.Header.disconnect()
		return
	}
}*/


