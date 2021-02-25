package server

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
	"udpserver/protocol"
)

type Connection struct {
	server         *Server
	Hostname       string
	IpAddress      *net.UDPAddr
	Domain         string
	Login          string
	ConnectTime    time.Time
	DisconnectTime *time.Time
	Version        string
	sync.RWMutex
	flagConnected bool
	ticker        *time.Ticker
}

func (c *Connection) startTimer(timeout int) {
	c.ticker = time.NewTicker(time.Duration(timeout) * time.Second)
	for _ = range c.ticker.C {
		//Проверяем на признак подключения
		if c.flagConnected {
			//Через timeout времени ставим флаг в неподключен
			c.Lock()
			c.flagConnected = false
			c.Unlock()
			continue
		} else {
			c.disconnect()
		}
	}
}

func (c *Connection) updated(addr *net.UDPAddr, header protocol.Header) bool {

	c.Lock()
	c.flagConnected = true
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
	if c == nil {
		return
	}
	c.ticker.Stop()
	c.flagConnected = false
	t := time.Now()
	c.DisconnectTime = &t

	//Удаляем подключение из списка
	c.server.deleteConnection(c.Hostname)
	//событие при отключении
	c.server.handleDisconnected(c)
}

func (c *Connection) send(response *protocol.Response) (n int, err error) {

	//Сериализуем данные в json
	sendMessage, err := json.Marshal(response)
	if err != nil {
		return
	}

	//Запускаем отправку
	remoteAddr, err := net.ResolveUDPAddr(Udp4, c.IpAddress.String())
	if err != nil {
		return
		//logger.Save("server", "send", err.Error())
	}
	conn, err := net.DialUDP(Udp4, nil, remoteAddr)
	if err != nil {
		return
	}

	defer conn.Close()

	n, err = conn.Write(sendMessage)
	if err != nil {
		return
	}

	return
}

func (c *Connection) String() string {
	disconnect_time := "null"
	if c.DisconnectTime != nil {
		disconnect_time = c.DisconnectTime.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("hostname: %s, ip: %s, domain: %s, login: %s, version: %s, is_connected: %t, connect_time: %s, disconnect_time: %s",
		c.Hostname, c.IpAddress.String(), c.Domain, c.Login, c.Version, c.flagConnected,
		c.ConnectTime.Format("2006-01-02 15:04:05"), disconnect_time)
}
