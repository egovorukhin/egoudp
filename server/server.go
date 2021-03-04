package server

import (
	"errors"
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const udp = "udp"

//События
type HandleConnected func(c *Connection)
type HandleDisconnected func(c *Connection)

type Server struct {
	Connections sync.Map //*Connections
	listener    *net.UDPConn
	*log.Logger
	Config
	Started            bool
	Router             sync.Map
	handleConnected    HandleConnected
	handleDisconnected HandleDisconnected
}

type Config struct {
	Port              int
	BufferSize        int
	DisconnectTimeOut int
	LogLevel          LogLevel
}

type LogLevel int

const (
	LogLevelLow LogLevel = iota
	LogLevelHigh
)

type IServer interface {
	GetConnections() map[string]*Connection
	GetRoutes() map[string]*Route
	SetLogger(out io.Writer, prefix string, flag int)
	Start() error
	Stop() error
	Send(route string, resp *protocol.Response) (int, error)
	SetRoute(path string, method protocol.Methods, handler FuncHandler)
	HandleConnected(handler HandleConnected)
	HandleDisconnected(handler HandleDisconnected)
}

func New(config Config) IServer {
	return &Server{
		Connections: sync.Map{},
		Config:      config,
		Started:     false,
		Logger:      log.New(os.Stdout, "", log.Ldate|log.Ltime),
	}
}

func (s *Server) SetLogger(out io.Writer, prefix string, flag int) {
	s.Logger = log.New(out, prefix, flag)
}

func (s *Server) Start() (err error) {

	localAddr, err := net.ResolveUDPAddr(udp, ":"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}

	s.listener, err = net.ListenUDP(udp, localAddr)
	if err != nil {
		return err
	}

	go func() {

		for {

			if !s.Started {
				break
			}

			buffer := make([]byte, s.BufferSize)

			n, addr, err := s.listener.ReadFromUDP(buffer)
			if err != nil {
				s.Println(err)
				continue
			}

			if s.LogLevel == LogLevelHigh {
				s.Println("%s(%d)", string(buffer[:n]), n)
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

	//Запускаем таймер который будет удалять
	//коннект при отсутствии прилетающих пакетов
	go connection.startTimer(s.DisconnectTimeOut)

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

	//Инициализируем ответ
	resp := protocol.NewResponse(packet.Request, packet.Header.Event)
	//Установка/проверка подключения
	conn := s.setConnection(addr, packet)

	//Проверяем события
	switch packet.Header.Event {
	//Отправляем команду о подключении клиенту
	case protocol.EventConnected:
		//событие подключения клиента
		s.handleConnected(conn)
		//отпраляем клиенту ответ
		go conn.Send(resp.OK(nil))
		return
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//удаляем подключения из списка
		conn.disconnect()
		//событие отключения клиента
		s.handleDisconnected(conn)
		//отпраляем клиенту ответ
		return
	}

	if packet.Request != nil {
		//Если есть данные с прицепом, то что то с ними делаем...
		go s.handleFuncRoute(conn, resp, *packet.Request)
	}
}

func (s *Server) setConnection(addr *net.UDPAddr, packet *protocol.Packet) (conn *Connection) {
	//Возвращаем подключение по имени компа
	v, ok := s.Connections.Load(packet.Header.Hostname)
	if !ok {
		//Создаем и добавляем подключение
		conn = s.newConnection(addr, packet.Header)
		s.Connections.Store(packet.Header.Hostname, conn)
		packet.Header.Event = protocol.EventConnected
		return conn
	}
	//Приводим значение из списка к Connection
	conn = v.(*Connection)

	//Если пришли немного отличающиеся данные,
	//то обновляем данные по подключению
	if conn.updated(addr, packet.Header) {
		packet.Header.Event = protocol.EventConnected
	}

	return conn
}

func (s *Server) SetRoute(path string, method protocol.Methods, handler FuncHandler) {
	s.Router.Store(path, &Route{
		Path:    path,
		Method:  method,
		Handler: handler,
	})
}

func (s *Server) handleFuncRoute(c *Connection, resp protocol.IResponse, req protocol.Request) {
	v, ok := s.Router.Load(req.Path)
	if ok {
		route := v.(*Route)
		if route.Method != req.Method {
			resp.Error([]byte(fmt.Sprintf("Метод запроса не соответствует маршруту [%s]", route.Path)))
			return
		}
		go route.Handler(c, resp, req)
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

func (s *Server) GetRoutes() (routes map[string]*Route) {
	s.Connections.Range(func(key, value interface{}) bool {
		routes[key.(string)] = value.(*Route)
		return true
	})
	return
}

func (s *Server) Stop() error {
	s.Started = false
	return s.listener.Close()
}
