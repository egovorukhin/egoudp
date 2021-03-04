package protocol

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	p1 := Packet{
		Header: Header{
			Hostname: "Computer",
			Login:    "user",
			Domain:   "HQ",
			Version:  "3.3.6",
			Event:    EventDisconnect,
		},
		Request: &Request{
			Path:        "example",
			Id:          "123456",
			Method:      MethodSet,
			ContentType: "json",
			Data:        []byte(`{"message": "Hello, world!"}`),
		},
	}
	b := p1.Marshal()
	fmt.Println(b)

	p := new(Packet)
	err := p.Unmarshal(b)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(p)

	if p.Request.ContentType == "json" {
		type Data struct {
			Message string `json:"message"`
		}
		data := Data{}
		err = json.Unmarshal(p.Request.Data, &data)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(data)
	}
}
