package server

import (
	"fmt"
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
	sync.RWMutex
	Connected            bool
	disconnectTimer      *time.Ticker
	checkConnectionTimer *time.Ticker
}

func (c *Connection) startDisconnectTimer(timeout int) {
	c.disconnectTimer = time.NewTicker(time.Duration(timeout) * time.Second)
	for _ = range c.disconnectTimer.C {
		c.Lock()
		if !c.Connected {
			c.disconnect()
			return
		}
		c.Connected = false
		c.Unlock()
	}
}

func (c *Connection) startCheckConnectionTimer(timeout int) {
	c.checkConnectionTimer = time.NewTicker(time.Duration(timeout) * time.Second)
	for _ = range c.checkConnectionTimer.C {
		c.Lock()
		if !c.Connected {
			return
		}
		c.Unlock()

		resp := &protocol.Response{
			StatusCode: protocol.StatusCodeOK,
			Event:      protocol.EventCheckConnection,
		}
		n, err := c.Send(resp)
		if err != nil {
			c.Println(err)
			continue
		}

		if c.LogLevel == LogLevelHigh {
			c.Printf("CheckConnection: %s(%d)\n", resp.String(), n)
		}
	}
}

func (c *Connection) updated(addr *net.UDPAddr, header protocol.Header) bool {

	c.Lock()
	c.Connected = true
	c.Unlock()

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
	c.disconnectTimer.Stop()
	c.checkConnectionTimer.Stop()
	t := time.Now()
	c.DisconnectTime = &t
	//c.UDPConn.Close()
	//Удаляем подключение из списка
	c.deleteConnection(c.Hostname)
	//событие при отключении
	if c.handleDisconnected != nil {
		c.handleDisconnected(c)
	}
}

func (c *Connection) Send(resp protocol.IResponse) (int, error) {
	if !c.Started {
		return 0, nil
	}
	return c.listener.WriteToUDP(resp.Marshal(), c.IpAddress)
}

func (c *Connection) String() string {
	disconnect_time := "null"
	if c.DisconnectTime != nil {
		disconnect_time = c.DisconnectTime.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("hostname: %s, ip: %s, domain: %s, login: %s, version: %s, is_connected: %t, connect_time: %s, disconnect_time: %s",
		c.Hostname, c.IpAddress.String(), c.Domain, c.Login, c.Version, c.Connected,
		c.ConnectTime.Format("2006-01-02 15:04:05"), disconnect_time)
}
