package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

type Data struct {
	ContentType ContentType `json:"content_type"`
	Body        interface{} `json:"body"`
}

type ContentType int

const (
	Text ContentType = iota
	Json
	Xml
	Yaml
)

func (c ContentType) Marshal(v interface{}) ([]byte, error) {
	switch c {
	case Json:
		return json.Marshal(v)
	case Xml:
		return xml.Marshal(v)
	default:
		return []byte(v.(string)), nil
	}
}

func SetData(contentType ContentType, body interface{}) *Data {
	return &Data{
		ContentType: contentType,
		Body:        body,
	}
}

func (d *Data) String() string {
	return fmt.Sprintf("content_type: %v, body: %v", d.ContentType, d.Body)
}
