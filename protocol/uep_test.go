package protocol

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	p1 := UEP{
		Header: Header{
			Hostname: "GB1-DIT-1-16146",
			Login:    "govorukhin_35893",
			Domain:   "HQ",
			Version:  "3.3.6",
			Event:    EventDisconnect,
		},
		Body:   &Body{
			Route:  "example",
			Id:     "123456",
			Method: MethodSet,
			Type:   "json",
			Data:   []byte(`{"message": "Hello, world!"}`),
		},
	}
	b := p1.Marshal()
	fmt.Println(b)

	p, err := Unmarshal(b)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(p)

	if p.Body.Type == "json" {
		type Data struct {
			Message string `json:"message"`
		}
		data := Data{}
		err = json.Unmarshal(p.Body.Data, &data)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(data)
	}
}
