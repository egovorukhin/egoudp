package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type UEP struct {
	Header   Header   `eup:"header"`
	Request  *Request `eup:"bodyChar,omitempty"`
	Response *Response
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
	2:id
	1:0-method
	4:type
	125:data
	$-endChar
*/

func (uep *UEP) Marshal() (b []byte) {
	buf := bytes.NewBuffer(b)
	buf.Write([]byte(string(startChar)))
	//header
	header := uep.Header
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Hostname), header.Hostname)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Login), header.Login)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Domain), header.Domain)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(header.Version), header.Version)))
	buf.Write([]byte(fmt.Sprintf("1:%d", header.Event)))	
	//bodyChar
	req := uep.Request
	buf.Write([]byte(string(bodyChar)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Route), req.Route)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Id), req.Id)))
	buf.Write([]byte(fmt.Sprintf("1:%d", req.Method)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Type), req.Type)))
	buf.Write([]byte(fmt.Sprintf("%d:%s", len(req.Data), req.Data)))
	buf.Write([]byte(string(endChar)))
	
	return buf.Bytes()

}

func Unmarshal(b []byte) (uep *UEP, err error) {

	uep = new(UEP)

	if b[0] != startChar {
		return nil, errors.New("Первый символ должен быть - ^")
	}
	if b[len(b)-1] != endChar {
		return nil, errors.New("Последний символ должен быть - ^")
	}
	//var err error
	//1. hostname
	uep.Header.Hostname, b, err = findField(b[1:])
	if err != nil {
		return nil, err
	}
	//2. login
	uep.Header.Login, b, err = findField(b)
	if err != nil {
		return nil, err
	}
	//3. domain
	uep.Header.Domain, b, err = findField(b)
	if err != nil {
		return nil, err
	}
	//4. version client
	uep.Header.Version, b, err = findField(b)
	if err != nil {
		return nil, err
	}
	//5. event
	event, b, err := findField(b)
	if err != nil {
		return nil, err
	}
	uep.Header.Event.unmarshal(event)

	//body
	if b[0] == bodyChar {

		body := new(Request)
		
		//1. route	
		body.Route, b, err = findField(b[1:])
		if err != nil {
			return nil, err
		}
		//2. id
		body.Id, b, err = findField(b)
		if err != nil {
			return nil, err
		}
		//3. method
		method, b, err := findField(b)
		if err != nil {
			return nil, err
		}
		body.Method.unmarshal(method)
		//4. type
		body.Type, b, err = findField(b)
		if err != nil {
			return nil, err
		}
		//4. data
		data, b, err := findField(b)
		if err != nil {
			return nil, err
		}
		body.Data = []byte(data)

		uep.Request = body
	}

	return uep, nil
}

func findField(b []byte) (string, []byte, error) {
	for i, value := range b {
		if value == ':' {
			n, err := strconv.Atoi(string(b[:i]))
			if err != nil {
				return "", b, err
			}
			return string(b[i+1:n+i+1]), b[n+i+1:], nil
		}
	}
	return "", b, errors.New("Не удалось определить поле. Формат должен быть вида - n:word")
}

func (uep *UEP) String() string {
	body := "null"
	if uep.Request != nil {
		body = fmt.Sprintf("{%s}", uep.Request.String())
	}
	return fmt.Sprintf("header: {%s}, body: %s", uep.Header.String(), body)
}
