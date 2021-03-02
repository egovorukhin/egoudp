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

const Udp4 = "udp4"

//События
type HandleConnected func(c *Connection)
type HandleDisconnected func(c *Connection)

//Функция которая вызывается при событии получения определённого маршоута
type FuncHandler func(c *Connection, resp protocol.IResponse, req protocol.Request)

type Server struct {
	Connections sync.Map //*Connections
	listener    *net.UDPConn
	Config      Config
	Started     bool
	Router      sync.Map
	//handleSaveConnection HandleSaveConnection
	handleConnected    HandleConnected
	handleDisconnected HandleDisconnected
}

type Config struct {
	LocalPort int
	//RemotePort        int
	BufferSize        int
	DisconnectTimeOut int
}

type IServer interface {
	GetConnections() map[string]*Connection
	GetRoutes() []string
	Start() error
	Stop() error
	Send(string, *protocol.Response) (int, error)
	SetRoute(string, FuncHandler)
	HandleConnected(handler HandleConnected)
	HandleDisconnected(handler HandleDisconnected)
}

func NewServer(config Config) IServer {
	return &Server{
		Connections: sync.Map{},
		Config:      config,
		Started:     false,
	}
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

			buffer := make([]byte, s.Config.BufferSize)

			n, addr, err := s.listener.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			//Передаем данные и разбираем их
			go s.handleBufferParse(addr, buffer[:n])
		}
	}()

	s.Started = true

	return
}

func (s *Server) newConnection(addr *net.UDPAddr, header protocol.Header) *Connection {

	connection := &Connection{
		Server:         s,
		Hostname:       header.Hostname,
		IpAddress:      addr,
		Domain:         header.Domain,
		Login:          header.Login,
		ConnectTime:    time.Now(),
		DisconnectTime: nil,
		Version:        header.Version,
		RWMutex:        sync.RWMutex{},
		flagConnected:  true,
		ticker:         nil,
	}
	/*
		if err := connection.SetUDPConn(addr); err != nil {
			return nil
		}*/

	//Запускаем таймер который будет удалять
	//коннект при отсутствии прилетающих пакетов
	go connection.startTimer(s.Config.DisconnectTimeOut)

	return connection
}

func (s *Server) deleteConnection(hostname string) {
	s.Connections.Delete(hostname)
}

func (s *Server) handleBufferParse(addr *net.UDPAddr, buffer []byte) {

	packet := new(protocol.Packet)
	err := packet.Unmarshal(buffer)
	if err != nil {
		return
	}

	if !packet.Header.IsNil() {
		//Приводим к правильному формату данные - эстетика
		packet.Header.Login = strings.ToLower(packet.Header.Login)
		packet.Header.Domain = strings.ToUpper(packet.Header.Domain)
		packet.Header.Hostname = strings.ToUpper(packet.Header.Hostname)

		//Подключаемся
		go s.receive(addr, packet)
	}
}

func (s *Server) receive(addr *net.UDPAddr, packet *protocol.Packet) {

	//Возвращаем подключение по имени компа
	v, ok := s.Connections.Load(packet.Header.Hostname)
	if !ok {
		//Создаем и добавляем подключение
		s.Connections.Store(packet.Header.Hostname, s.newConnection(addr, packet.Header))
		return
	}
	connection := v.(*Connection)

	//Обновляем данные по подключению
	if connection.updated(addr, packet.Header) {
		go s.handleConnected(connection)
	}

	resp := protocol.NewResponse(packet.Request, packet.Header.Event)

	//Проверяем события
	switch packet.Header.Event {
	//Отправляем команду о подключении клиенту
	case protocol.EventConnected:
		//событие подключения клиента
		s.handleConnected(connection)
		//отпраляем клиенту ответ
		go connection.Send(resp.OK(nil))
		return
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//удаляем подключения из списка
		connection.disconnect()
		//событие отключения клиента
		s.handleDisconnected(connection)
		//отпраляем клиенту ответ
		//go connection.Send(resp.OK(nil))
		return
	}

	if packet.Request != nil {
		//Если есть данные с прицепом, то что то с ними делаем...
		s.handleFuncRoute(connection, resp, *packet.Request)
	}
}

func (s *Server) SetRoute(route string, handler FuncHandler) {
	s.Router.Store(route, handler)
}

func (s *Server) handleFuncRoute(c *Connection, resp protocol.IResponse, req protocol.Request) {
	v, ok := s.Router.Load(req.Route)
	if !ok {
		go v.(FuncHandler)(c, resp, req)
	}
}

func (s *Server) HandleConnected(handler HandleConnected) {
	s.handleConnected = handler
}

func (s *Server) HandleDisconnected(handler HandleDisconnected) {
	s.handleDisconnected = handler
}

func (s *Server) Send(hostname string, response *protocol.Response) (n int, err error) {

	//Проверяем на существование подключение
	v, ok := s.Connections.Load(hostname)
	if !ok {
		return 0, errors.New(fmt.Sprintf("host: %s - подключение отсутствует!", hostname))
	}

	connection := v.(*Connection)

	return connection.Send(response)
}

func (s *Server) GetConnections() (connections map[string]*Connection) {
	s.Connections.Range(func(key, value interface{}) bool {
		connections[key.(string)] = value.(*Connection)
		return true
	})
	return
}

func (s *Server) GetRoutes() (routes []string) {
	s.Connections.Range(func(key, value interface{}) bool {
		routes = append(routes, key.(string))
		return true
	})
	return
}

func (s *Server) Stop() error {
	s.Started = false
	return s.listener.Close()
}
