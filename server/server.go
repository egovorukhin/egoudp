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

type Server struct {
	Connections sync.Map
	listener    *net.UDPConn
	Started     Started
	Router      sync.Map
	Handler     *Handler
	*log.Logger
	Config
}

type Config struct {
	Port                   int
	BufferSize             int
	DisconnectTimeout      int
	CheckConnectionTimeout int
	LogLevel               LogLevel
}

type Started struct {
	sync.Mutex
	value bool
}

func (c *Started) Set(b bool) {
	c.Lock()
	c.value = b
	c.Unlock()
}

func (c *Started) Get() bool {
	c.Lock()
	defer c.Unlock()
	return c.value
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
	Send(hostname string, resp *protocol.Response) (int, error)
	SendByLogin(login string, response *protocol.Response) int
	SetRoute(path string, method protocol.Methods, handler FuncHandler)
	OnStart(handler HandleServer)
	OnStop(handler HandleServer)
	OnConnected(handler HandleConnection)
	OnDisconnected(handler HandleConnection)
}

func New(config Config) IServer {
	return &Server{
		Connections: sync.Map{},
		Config:      config,
		Started:     Started{},
		Logger:      log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Handler:     new(Handler),
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

	go s.receive()

	s.Started.Set(true)

	OnStart(s.Handler, s)

	return
}

func (s *Server) receive() {

	for {

		buffer := make([]byte, s.BufferSize)

		n, addr, err := s.listener.ReadFromUDP(buffer)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			s.Printf("receive: %v\n", err)
			continue
		}

		if !s.Started.Get() {
			break
		}

		if s.LogLevel == LogLevelHigh {
			s.Println("receive: %s(%d)", string(buffer[:n]), n)
		}

		//Передаем данные и разбираем их
		go s.parse(addr, buffer[:n])
	}
}

func (s *Server) newConnection(addr *net.UDPAddr, header protocol.Header) *Connection {

	conn := &Connection{
		Server:      s,
		Hostname:    header.Hostname,
		IpAddress:   addr,
		Domain:      header.Domain,
		Login:       header.Login,
		ConnectTime: time.Now(),
		Version:     header.Version,
		Connected: Connected{
			value: true,
		},
	}

	//Запускаем таймер который будет удалять коннект
	//при отсутствии прилетающих пакетов
	conn.startDTimer(s.DisconnectTimeout)
	//Таймер отправки активности сервера
	//conn.startCCTimer(s.CheckConnectionTimeout)

	return conn
}

func (s *Server) deleteConnection(hostname string) {
	s.Connections.Delete(hostname)
}

func (s *Server) parse(addr *net.UDPAddr, buffer []byte) {

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
		go s.do(addr, packet)
	}
}

func (s *Server) do(addr *net.UDPAddr, packet *protocol.Packet) {

	//Инициализируем ответ
	resp := protocol.NewResponse(packet.Request, packet.Header.Event)
	//Установка/проверка подключения
	conn := s.setConnection(addr, packet)

	//Проверяем события
	switch packet.Header.Event {
	//Отправляем команду о подключении клиенту
	case int(protocol.EventConnected):
		//отправляем клиенту ответ
		go conn.Send4(int(protocol.EventConnected))
		return
	//Команда на отключение клиента
	case int(protocol.EventDisconnect):
		//удаляем подключения из списка
		conn.Connected.Set(false)
		conn.disconnect()
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
		//событие подключения клиента
		OnConnected(s.Handler, conn)
		packet.Header.Event = int(protocol.EventConnected)
		return conn
	}
	//Приводим значение из списка к Connection
	conn = v.(*Connection)

	//Если пришли немного отличающиеся данные,
	//то обновляем данные по подключению
	if conn.updated(addr, packet.Header) {
		//событие переподключения клиента
		OnReconnected(s.Handler, conn)
		packet.Header.Event = int(protocol.EventConnected)
	}

	conn.Connected.Set(true)

	return conn
}

func (s *Server) SetRoute(path string, method protocol.Methods, handler FuncHandler) {
	s.Router.Store(fmt.Sprintf("%s:%d", path, method), &Route{
		Path:    path,
		Method:  method,
		Handler: handler,
	})
}

func (s *Server) handleFuncRoute(c *Connection, resp protocol.IResponse, req protocol.Request) {
	v, ok := s.Router.Load(fmt.Sprintf("%s:%d", req.Path, req.Method))
	if ok {
		route := v.(*Route)
		if route.Method != req.Method {
			resp.SetData(protocol.StatusCodeError, []rune(fmt.Sprintf("Метод запроса не соответствует маршруту [%s]", route.Path)))
			return
		}
		go route.Handler(c, resp, req)
	}
}

func (s *Server) Send(hostname string, response *protocol.Response) (n int, err error) {

	//Проверяем на существование подключение
	v, ok := s.Connections.Load(strings.ToUpper(hostname))
	if !ok {
		return 0, errors.New(fmt.Sprintf("host: %s - подключение отсутствует!", hostname))
	}

	connection := v.(*Connection)

	return connection.Send(response)
}

func (s *Server) SendByLogin(login string, response *protocol.Response) (n int) {
	//Ищем по логину тачки
	s.Connections.Range(func(key, value interface{}) bool {
		connection := value.(*Connection)
		if connection.Login == strings.ToLower(login) {
			_, _ = connection.Send(response)
			n++
		}
		return true
	})
	return
}

func (s *Server) GetConnections() (connections map[string]*Connection) {
	connections = map[string]*Connection{}
	s.Connections.Range(func(key, value interface{}) bool {
		connections[key.(string)] = value.(*Connection)
		return true
	})
	return
}

func (s *Server) GetRoutes() (routes map[string]*Route) {
	routes = map[string]*Route{}
	s.Router.Range(func(key, value interface{}) bool {
		routes[key.(string)] = value.(*Route)
		return true
	})
	return
}

func (s *Server) OnStart(handler HandleServer) {
	s.Handler.OnStart = handler
}

func (s *Server) OnStop(handler HandleServer) {
	s.Handler.OnStop = handler
}

func (s *Server) OnConnected(handler HandleConnection) {
	s.Handler.OnConnected = handler
}

func (s *Server) OnDisconnected(handler HandleConnection) {
	s.Handler.OnDisconnected = handler
}

func (s *Server) Stop() error {
	//defer s.listener.Close()
	OnStop(s.Handler, s)
	s.Started.Set(false)
	for _, conn := range s.GetConnections() {
		conn.Connected.Set(false)
		conn.disconnect()
	}
	return s.listener.Close()
}
