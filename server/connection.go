package server

import (
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

func (connection *Connection) startTimer(timeout int) {
	connection.ticker = time.NewTicker(time.Duration(timeout) * time.Second)
	for _ = range connection.ticker.C {
		//Проверяем на признак подключения
		if connection.flagConnected {
			//Через timeout времени ставим флаг в неподключен
			connection.Lock()
			connection.flagConnected = false
			connection.Unlock()
			continue
		} else {
			connection.server.disconnect()
		}
	}
}

func (connection *Connection) update(addr *net.UDPAddr, header protocol.Header) /*(err error)*/ {

	connection.Lock()
	connection.flagConnected = true
	connection.Unlock()

	if !connection.Equals(header) || !strings.EqualFold(connection.IpAddress.String(), addr.String()) /*!c.IpAddress.IP.Equal(addr.IP)*/ {
		connection.Hostname = header.Hostname
		connection.IpAddress = addr
		connection.Domain = header.Domain
		connection.Login = header.Login
		connection.Version = header.Version

		go connection.saveConnection()
	}
}

func (connection *Connection) Equals(header protocol.Header) bool {
	if connection.Hostname != header.Hostname ||
		connection.Login != header.Login ||
		connection.Domain != header.Domain ||
		connection.Version != header.Version {
		return false
	}
	return true
}

func (connection *Connection) disconnect() {
	if connection == nil {
		return
	}
	connection.ticker.Stop()
	connection.flagConnected = false
	t := time.Now()
	connection.DisconnectTime = &t

	go connection.saveConnection()

	connection.server.Connections.Delete(connection.Hostname)
}

func (connection *Connection) saveConnection() {
	handleSaveConnection(
		connection.Hostname,
		connection.IpAddress.IP.String(),
		connection.Login,
		connection.Domain,
		connection.Version,
		connection.flagConnected,
		connection.ConnectTime,
		connection.DisconnectTime)
}

func (connection *Connection) String() string {
	disconnect_time := "null"
	if connection.DisconnectTime != nil {
		disconnect_time = connection.DisconnectTime.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("hostname: %s, ip: %s, domain: %s, login: %s, version: %s, is_connected: %t, connect_time: %s, disconnect_time: %s",
		connection.Hostname, connection.IpAddress.String(), connection.Domain, connection.Login, connection.Version, connection.flagConnected,
		connection.ConnectTime.Format("2006-01-02 15:04:05"), disconnect_time)
}
