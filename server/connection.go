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
	dTimer         *egotimer.Timer
	ccTimer        *egotimer.Timer
	Connected      Connected
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
	c.dTimer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		if !c.Connected.Get() {
			c.disconnect()
			return true
		}
		c.Connected.Set(false)
		return false
	})
	go c.dTimer.Start()
}

//Стартуем check connection timer
func (c *Connection) startCCTimer(timeout int) {
	c.ccTimer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		c.SendEvent(protocol.EventCheckConnection)
		return false
	})
	go c.ccTimer.Start()
}

func (c *Connection) updated(addr *net.UDPAddr, header protocol.Header) bool {

	if !c.Equals(header) || !strings.EqualFold(c.IpAddress.String(), addr.String()) /*!c.IpAddress.IP.Equal(addr.IP)*/ {
		c.Hostname = header.Hostname
		c.IpAddress = addr
		c.Domain = header.Domain
		c.Login = header.Login
		c.Version = header.Version

		return true
	}

	return false
}

func (c *Connection) Equals(header protocol.Header) bool {
	if c.Hostname != header.Hostname ||
		c.Login != header.Login ||
		c.Domain != header.Domain ||
		c.Version != header.Version {
		return false
	}
	return true
}

func (c *Connection) disconnect() {
	c.dTimer.Stop()
	c.ccTimer.Stop()
	c.SendEvent(protocol.EventDisconnect)
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

func (c *Connection) SendEvent(event protocol.Events) {
	resp := &protocol.Response{
		StatusCode: protocol.StatusCodeOK,
		Event:      event,
	}
	n, err := c.Send(resp)
	if err != nil {
		c.Println(err)
		return
	}
	if c.LogLevel == LogLevelHigh {
		c.Printf("CheckConnection: %s(%d)\n", resp.String(), n)
	}
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
