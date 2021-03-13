package server

import (
	"fmt"
	"github.com/egovorukhin/egotimer"
	"github.com/egovorukhin/egoudp/protocol"
	"net"
	"strings"
	"sync"
	"time"
)

type Connection struct {
	*Server
	Hostname       string
	IpAddress      *net.UDPAddr
	Domain         string
	Login          string
	ConnectTime    time.Time
	DisconnectTime *time.Time
	Version        string
	timer          *egotimer.Timer
	//ccTimer        *egotimer.Timer
	Connected Connected
}

type Connected struct {
	sync.Mutex
	value bool
}

func (c *Connected) Set(b bool) {
	c.Lock()
	c.value = b
	c.Unlock()
}

func (c *Connected) Get() bool {
	c.Lock()
	defer c.Unlock()
	return c.value
}

func (c *Connection) startDTimer(timeout int) {
	c.timer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		if !c.Connected.Get() {
			c.disconnect()
			return true
		}
		c.Connected.Set(false)
		return false
	})
	go c.timer.Start()
}

//Стартуем check connection timer
/*func (c *Connection) startCCTimer(timeout int) {
	c.ccTimer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		c.Send4(protocol.EventCheckConnection)
		return false
	})
	go c.ccTimer.Start()
}*/

func (c *Connection) updated(addr *net.UDPAddr, header protocol.Header) bool {

	if !c.equals(header) || !strings.EqualFold(c.IpAddress.String(), addr.String()) /*!c.IpAddress.IP.Equal(addr.IP)*/ {
		c.Hostname = header.Hostname
		c.IpAddress = addr
		c.Domain = header.Domain
		c.Login = header.Login
		c.Version = header.Version

		return true
	}

	return false
}

func (c *Connection) equals(header protocol.Header) bool {
	if c.Hostname != header.Hostname ||
		c.Login != header.Login ||
		c.Domain != header.Domain ||
		c.Version != header.Version {
		return false
	}
	return true
}

func (c *Connection) disconnect() {
	c.timer.Stop()
	//c.ccTimer.Stop()
	c.Send4(int(protocol.EventDisconnect))
	t := time.Now()
	c.DisconnectTime = &t
	//Удаляем подключение из списка
	c.deleteConnection(c.Hostname)
	//событие при отключении
	OnDisconnected(c.Handler, c)
}

func (c *Connection) Send(resp protocol.IResponse) (int, error) {
	return c.listener.WriteToUDP(resp.Marshal(), c.IpAddress)
}

func (c *Connection) Send1(resp *protocol.Response) {
	n, err := c.Send(resp)
	if err != nil {
		c.Printf("Send1: %v\n", err)
		return
	}
	if c.LogLevel == LogLevelHigh {
		c.Printf("Send1: %s(%d)\n", resp.String(), n)
	}
}

func (c *Connection) Send2(code protocol.StatusCode, event int, contentType string, data []rune) {
	resp := &protocol.Response{
		StatusCode:  code,
		Event:       event,
		ContentType: contentType,
		Data:        data,
	}
	c.Send1(resp)
}

func (c *Connection) Send3(event int, contentType string, data []rune) {
	c.Send2(protocol.StatusCodeOK, event, contentType, data)
}

func (c *Connection) Send4(event int) {
	c.Send3(event, "", nil)
}

func (c *Connection) String() string {
	disconnect_time := "null"
	if c.DisconnectTime != nil {
		disconnect_time = c.DisconnectTime.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("hostname: %s, ip: %s, domain: %s, login: %s, version: %s, connected: %t, connect_time: %s, disconnect_time: %s",
		c.Hostname, c.IpAddress.String(), c.Domain, c.Login, c.Version, c.Connected.value,
		c.ConnectTime.Format("2006-01-02 15:04:05"), disconnect_time)
}
