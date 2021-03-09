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
type HandleServer func(s *Server)
type HandleConnection func(c *Connection)

type Server struct {
	Connections sync.Map //*Connections
	listener    *net.UDPConn
	*log.Logger
	Config
	Started            bool
	Router             sync.Map
	handleStart        HandleServer
	handleStop         HandleServer
	handleConnected    HandleConnection
	handleDisconnected HandleConnection
}

type Config struct {
	Port                   int
	BufferSize             int
	DisconnectTimeout      int
	CheckConnectionTimeout int
	LogLevel               LogLevel
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
	Stop()
	Send(route string, resp *protocol.Response) (int, error)
	SetRoute(path string, method protocol.Methods, handler FuncHandler)
	HandleStart(handler HandleServer)
	HandleStop(handler HandleServer)
	HandleConnected(handler HandleConnection)
	HandleDisconnected(handler HandleConnection)
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

		defer s.listener.Close()

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
			/*bytes := 0
			var addr *net.UDPAddr
			var data []byte
			for bytes > 0 {
				var n int
				n, addr, err = s.listener.ReadFromUDP(buffer)
				if err != nil {
					s.Println(err)
					break
				}
				data = append(data, buffer[:n]...)
			}*/

			if s.LogLevel == LogLevelHigh {
				s.Println("%s(%d)", string(buffer[:n]), n)
			}

			//Передаем данные и разбираем их
			go s.handleBufferParse(addr, buffer[:n])
		}
	}()

	s.Started = true

	if s.handleStart != nil {
		go s.handleStart(s)
	}

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
		Connected:      true,
		dTimer:         nil,
	}
	//Запускаем таймер который будет удалять
	//коннект при отсутствии прилетающих пакетов
	//connection.done = make(chan bool)
	go connection.startDTimer(s.DisconnectTimeout)
	//Таймер отправки активности сервера
	go connection.startCCTimer(s.CheckConnectionTimeout)

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

		if !conn.Connected {
			return
		}

		//событие подключения клиента
		if s.handleConnected != nil {
			go s.handleConnected(conn)
		}
		//отправляем клиенту ответ,
		//передаем время для таймера проверки активности сервера
		//прибавляем 5 сек, чтобы расклиент ждал проверку дольше
		go func() {
			_, err := conn.Send(resp.OK([]byte(strconv.Itoa(s.CheckConnectionTimeout + 5))))
			if err != nil {
				s.Println(err)
			}
		}()
		return
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//удаляем подключения из списка
		//conn.disconnect()
		conn.Lock()
		conn.Connected = false
		conn.Unlock()
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

	conn.Lock()
	conn.Connected = true
	conn.Unlock()

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

func (s *Server) HandleStart(handler HandleServer) {
	s.handleStart = handler
}

func (s *Server) HandleStop(handler HandleServer) {
	s.handleStop = handler
}

func (s *Server) HandleConnected(handler HandleConnection) {
	s.handleConnected = handler
}

func (s *Server) HandleDisconnected(handler HandleConnection) {
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

func (s *Server) Stop() {
	s.Started = false
	if s.handleStop != nil {
		s.handleStop(s)
	}
	for _, conn := range s.GetConnections() {
		conn.Lock()
		conn.Connected = false
		conn.Unlock()
		resp := &protocol.Response{
			StatusCode: protocol.StatusCodeOK,
			Event:      protocol.EventDisconnect,
		}
		n, err := conn.Send(resp)
		if err != nil {
			conn.Println(err)
			continue
		}
		if conn.LogLevel == LogLevelHigh {
			conn.Printf("CheckConnection: %s(%d)\n", resp.String(), n)
		}
	}
}
