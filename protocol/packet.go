package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type Packet struct {
	sync.Mutex
	Header
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

func New(hostname, login, domain, version string) *Packet {
	return &Packet{
		Header: Header{
			Hostname: hostname,
			Login:    login,
			Domain:   domain,
			Version:  version,
			Event:    int(EventNone),
		},
	}
}

func (p *Packet) SetEvent(event int) {
	p.Lock()
	p.Event = event
	p.Unlock()
}

func (p *Packet) GetEvent() int {
	p.Lock()
	defer p.Unlock()
	return p.Event
}

func (p *Packet) Marshal() (b []byte) {
	buf := bytes.NewBuffer(b)
	buf.Write([]byte(string(startChar)))
	//header
	header := p.Header
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Hostname), header.Hostname)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Login), header.Login)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Domain), header.Domain)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Version), header.Version)))
	buf.Write([]byte(fmt.Sprintf("1:%d", header.Event)))
	//bodyChar
	if p.Request != nil {
		req := p.Request
		buf.Write([]byte(string(bodyChar)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Path), req.Path)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Id), req.Id)))
		buf.Write([]byte(fmt.Sprintf("1:%d", req.Method)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.ContentType), req.ContentType)))
		buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Data), req.Data)))
	}
	buf.Write([]byte(string(endChar)))

	return buf.Bytes()

}

func (p *Packet) Unmarshal(b []byte) error {

	if b[0] != startChar {
		return errors.New(fmt.Sprintf("Первый символ должен быть - %v", startChar))
	}
	if b[len(b)-1] != endChar {
		return errors.New(fmt.Sprintf("Последний символ должен быть - %v", endChar))
	}
	var err error
	//1. hostname
	p.Header.Hostname, b, err = findField(b[1:])
	if err != nil {
		return err
	}
	//2. login
	p.Header.Login, b, err = findField(b)
	if err != nil {
		return err
	}
	//3. domain
	p.Header.Domain, b, err = findField(b)
	if err != nil {
		return err
	}
	//4. version client
	p.Header.Version, b, err = findField(b)
	if err != nil {
		return err
	}
	//5. event
	event, b, err := findField(b)
	if err != nil {
		return err
	}
	p.Header.Event, _ = strconv.Atoi(event)

	//body
	if b[0] == bodyChar {

		req := new(Request)

		//1. route
		req.Path, b, err = findField(b[1:])
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
		req.Data = []rune(data)

		p.Request = req
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
			r := ToRunes(string(b))
			return r[i+1 : n+i+1].String(), b[n+i+1:], nil
		}
	}
	return "", b, errors.New("Не удалось определить поле. Формат должен быть вида - n:word")
}

func (p *Packet) String() string {
	req := "null"
	if p.Request != nil {
		req = fmt.Sprintf("{%s}", p.Request.String())
	}
	/*resp := "null"
	if p.Response != nil {
		resp = fmt.Sprintf("{%s}", p.Response.String())
	}*/
	return fmt.Sprintf("header: {%s}, request: %s", p.Header.String(), req)
}
