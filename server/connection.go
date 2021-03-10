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
	sync.Mutex
	Connected bool
}

func (c *Connection) startDTimer(timeout int) {
	c.dTimer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		c.Lock()
		if !c.Connected {
			c.disconnect()
			return true
		}
		c.Connected = false
		c.Unlock()
		return false
	})
	c.dTimer.Start()
}

//Стартуем check connection timer
func (c *Connection) startCCTimer(timeout int) {
	c.ccTimer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		/*c.RLock()
		if !c.Connected {
			return true
		}
		c.RUnlock()*/

		resp := &protocol.Response{
			StatusCode: protocol.StatusCodeOK,
			Event:      protocol.EventCheckConnection,
		}
		n, err := c.Send(resp)
		if err != nil {
			c.Println(err)
			return false
		}

		if c.LogLevel == LogLevelHigh {
			c.Printf("CheckConnection: %s(%d)\n", resp.String(), n)
		}
		return false
	})
	c.ccTimer.Start()
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
