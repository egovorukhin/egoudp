package protocol

import "fmt"

type Request struct {
	Route  string
	Id     string  `eup:"id"`
	Method Methods `eup:"method"`
	Type   string  `eup:"type"`
	Data   []byte  `eup:"data,omitempty"`
}

func (body *Request) String() string {
	data := "null"
	if body.Data != nil {
		data = fmt.Sprintf("%v", body.Data)
	}
	return fmt.Sprintf("id: %s, method: %s, type: %s, data: %s", body.Id, body.Method.String(), body.Type, data)
}
