package server

import (
	"errors"
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

//var log = egologger.New(nil, "server")

const Udp4 = "udp4"

//События
/*type HandleSaveConnection func(hostname string, ipaddress string, login string, domain string, version string,
	connected bool, connectTime time.Time, disconnectTime *time.Time)*/
type HandleConnected func(c *Connection)
type HandleDisconnected func(c *Connection)

//Функция которая вызывает при событии получения определённого маршоута
type FuncHandler func(resp *protocol.Response, req protocol.Request)

type Server struct {
	Connections          sync.Map //*Connections
	listener             *net.UDPConn
	Config               Config
	Started              bool
	router               sync.Map
	//handleSaveConnection HandleSaveConnection
	handleConnected HandleConnected
	handleDisconnected HandleDisconnected
}

type Config struct {
	LocalPort         int
	RemotePort        int
	BufferSize        int
	DisconnectTimeOut int
}

type IServer interface {
	GetConnections() map[string]*Connection
	Start() error
	Stop() error
	Send(string, *protocol.Response) (int, error)
	//HandleSaveConnection(HandleSaveConnection)
	SetRoute(string, FuncHandler)
}

func NewServer(config Config) IServer {
	return &Server{
		Connections: sync.Map{},
		Config:      config,
		Started:     false,
	}
}
/*
func (s *Server) HandleSaveConnection(handler HandleSaveConnection) {
	s.handleSaveConnection = handler
}*/

func (s *Server) GetConnections() (connections map[string]*Connection) {
	s.Connections.Range(func(key, value interface{}) bool {
		connections[key.(string)] = value.(*Connection)
		return true
	})
	return
}

func (s *Server) Start() (err error) {

	localAddr, err := net.ResolveUDPAddr(Udp4, ":"+strconv.Itoa(s.Config.LocalPort))
	if err != nil {
		return err
	}

	s.listener, err = net.ListenUDP(Udp4, localAddr)
	if err != nil {
		return err
	}

	go func() {

		for {

			if !s.Started {
				break
			}

			buffer := make([]byte, 1024* s.Config.BufferSize)

			n, addr, err := s.listener.ReadFromUDP(buffer)
			if err != nil {
				//log.Error(fmt.Sprintf("listen %s: %s", addr, err.Error()))
				continue
			}

			//Передаем данные и разбираем их
			go s.handleBufferParse(addr, buffer[:n])
		}
	}()

	s.Started = true

	return
}

func (s *Server) Stop() error {
	s.Started = false
	return s.listener.Close()
}

func (s *Server) newConnection(addr *net.UDPAddr, header protocol.Header) *Connection {

	connection := &Connection{
		server:        s,
		Hostname:      header.Hostname,
		IpAddress:     addr,
		Domain:        header.Domain,
		Login:         header.Login,
		ConnectTime:   time.Now(),
		Version:       header.Version,
		flagConnected: true,
	}

	//Запускаем таймер который будет удалять
	//коннект при отсутствии прилетающих пакетов
	go connection.startTimer(s.Config.DisconnectTimeOut)

	//go s.saveConnection(connection)

	return connection
}

func (s *Server) deleteConnection(hostname string)  {
	s.Connections.Delete(hostname)
}

func (s *Server) handleBufferParse(addr *net.UDPAddr, buffer []byte) {

	message, err := protocol.Unmarshal(buffer)
	if err != nil {
		return
	}

	if !message.Header.IsNil() {
		//Приводим к правильному формату данные - эстетика
		message.Header.Login = strings.ToLower(message.Header.Login)
		message.Header.Domain = strings.ToUpper(message.Header.Domain)
		message.Header.Hostname = strings.ToUpper(message.Header.Hostname)

		//Подключаемся
		go s.receive(addr, message)
	}
}

func (s *Server) receive(addr *net.UDPAddr, uep *protocol.UEP) {

	//Возвращаем подключение по имени компа
	v, ok := s.Connections.Load(uep.Header.Hostname)
	if !ok {
		//Создаем и добавляем подключение
		s.Connections.Store(uep.Header.Hostname, s.newConnection(addr, uep.Header))
		return
	}
	connection := v.(*Connection)

	//Обновляем данные по подключению
	if connection.updated(addr, uep.Header) {
		go s.handleConnected(connection)
	}

	//Проверяем события
	switch uep.Header.Event {
	//Отправляем команду о подключении клиенту
	case protocol.EventConnected:
		//событие подключения клиента
		s.handleConnected(connection)
		//отпраляем клиенту ответ
		go connection.send(uep.Response)
		//response.OK(r.Header.Hostname, nil)
		break
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//удаляем подключения из списка
		connection.disconnect()
		//отпраляем клиенту ответ
		go connection.send(uep.Response)
		break
	default:

	}

	if uep.Request != nil {
		//Если есть данные с прицепом, то что то с ними делаем...
		s.handleFuncRoute(uep)
	}
}

func (s *Server) SetRoute(route string, handler FuncHandler) {
	s.router.Store(route, handler)
}

func (s *Server) handleFuncRoute(uep *protocol.UEP) {
	v, ok := s.router.Load(uep.Header.Hostname)
	if !ok {
		go v.(FuncHandler)(uep.Response, *uep.Request)
	}
}

/*
func (s *Server) saveConnection(c *Connection) {
	s.handleSaveConnection(
		c.Hostname,
		c.IpAddress.IP.String(),
		c.Login,
		c.Domain,
		c.Version,
		c.flagConnected,
		c.ConnectTime,
		c.DisconnectTime)
}*/

/*
func SendByLogin(login string, response *Response) (n int, err error) {
	chLength := make(chan int)
	chErr := make(chan error, 1)
	go func() {
		connections := srv.Connections.Find("Login", strings.ToLower(login))
		for _, connection := range connections {
			err := send(connection.Hostname, response)
			if err != nil {
				chLength <- 0
				chErr <- err
				return
			}
		}
		chLength <- len(connections)
		chErr <- nil
	}()
	return <-chLength, <-chErr
}

func (s *server) SendByHostname(hostname string, response *Response) (err error) {
	chErr := make(chan error)
	go func() {
		err := s.send(hostname, response)
		if err != nil {
			chErr <- err
			return
		}
		chErr <- nil
	}()
	return <-chErr
}*/

func (s *Server) Send(hostname string, response *protocol.Response) (n int, err error) {

	//Проверяем на существование подключение
	v, ok := s.Connections.Load(hostname)
	if !ok {
		return 0, errors.New(fmt.Sprintf("host: %s - подключение отсутствует!", hostname))
	}

	connection := v.(*Connection)

	return connection.send(response)
}
