package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type Packet struct {
	Header  Header
	Request *Request
}

const (
	startChar byte = '^'
	bodyChar       = '#'
	endChar        = '$'
)

/*
	^-startChar
	8:hostname
	5:login
	5:domain
	7:version
	e:0-event
	#-bodyChar
	5:route
	2:Id
	1:0-method
	4:type
	125:data
	$-endChar
*/

func NewUEP(hostname, login, domain, version string) *Packet {
	return &Packet{
		Header: Header{
			Hostname: hostname,
			Login:    login,
			Domain:   domain,
			Version:  version,
			Event:    EventNone,
		},
	}
}

func (r *Packet) Marshal() (b []byte) {
	buf := bytes.NewBuffer(b)
	buf.Write([]byte(string(startChar)))
	//header
	header := r.Header
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Hostname), header.Hostname)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Login), header.Login)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Domain), header.Domain)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Version), header.Version)))
	buf.Write([]byte(fmt.Sprintf("1:%d", header.Event)))
	//bodyChar
	if r.Request != nil {
		req := r.Request
		buf.Write([]byte(string(bodyChar)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Route), req.Route)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Id), req.Id)))
		buf.Write([]byte(fmt.Sprintf("1:%d", req.Method)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.ContentType), req.ContentType)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Data), req.Data)))
	}
	buf.Write([]byte(string(endChar)))

	return buf.Bytes()

}

func (r *Packet) Unmarshal(b []byte) error {

	if b[0] != startChar {
		return errors.New(fmt.Sprintf("Первый символ должен быть - %v", startChar))
	}
	if b[len(b)-1] != endChar {
		return errors.New(fmt.Sprintf("Последний символ должен быть - %v", endChar))
	}
	var err error
	//1. hostname
	r.Header.Hostname, b, err = findField(b[1:])
	if err != nil {
		return err
	}
	//2. login
	r.Header.Login, b, err = findField(b)
	if err != nil {
		return err
	}
	//3. domain
	r.Header.Domain, b, err = findField(b)
	if err != nil {
		return err
	}
	//4. version client
	r.Header.Version, b, err = findField(b)
	if err != nil {
		return err
	}
	//5. event
	event, b, err := findField(b)
	if err != nil {
		return err
	}
	r.Header.Event = ToEvent(event)

	//body
	if b[0] == bodyChar {

		req := new(Request)

		//1. route
		req.Route, b, err = findField(b[1:])
		if err != nil {
			return err
		}
		//2. Id
		req.Id, b, err = findField(b)
		if err != nil {
			return err
		}
		//3. method
		method, b, err := findField(b)
		if err != nil {
			return err
		}
		req.Method = ToMethod(method)
		//4. type
		req.ContentType, b, err = findField(b)
		if err != nil {
			return err
		}
		//4. data
		data, b, err := findField(b)
		if err != nil {
			return err
		}
		req.Data = []byte(data)

		r.Request = req
	}

	return nil
}

func findField(b []byte) (string, []byte, error) {
	for i, value := range b {
		if value == ':' {
			n, err := strconv.Atoi(string(b[:i]))
			if err != nil {
				return "", b, err
			}
			return string(b[i+1 : n+i+1]), b[n+i+1:], nil
		}
	}
	return "", b, errors.New("Не удалось определить поле. Формат должен быть вида - n:word")
}

func (r *Packet) String() string {
	req := "null"
	if r.Request != nil {
		req = fmt.Sprintf("{%s}", r.Request.String())
	}
	/*resp := "null"
	if r.Response != nil {
		resp = fmt.Sprintf("{%s}", r.Response.String())
	}*/
	return fmt.Sprintf("header: {%s}, request: %s", r.Header.String(), req)
}
