package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/egovorukhin/egologger"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"udpserver/protocol"
)

var log = egologger.New(nil, "server")

const Udp4 = "udp4"

type HandleSaveConnection func(hostname string, ipaddress string, login string, domain string, version string,
	connected bool, connectTime time.Time, disconnectTime *time.Time)

type Server struct {
	Connections sync.Map //*Connections
	listener    *net.UDPConn
	Config      Config
	Started     bool
	handleSaveConnection HandleSaveConnection
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
	Send(string, *Response) (int, error)
	HandleSaveConnection(HandleSaveConnection)
}

type ResponseWriter interface {
	Write([]byte) (int, error)
}

type Handler interface {
	Serve(ResponseWriter, *Request)
}

func NewServer(config Config) IServer {
	return &Server{
		Connections: sync.Map{},
		Config:      config,
		Started:     false,
	}
}

func (s *Server) HandleSaveConnection(handler HandleSaveConnection) {
	s.handleSaveConnection = handler
}

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

func (s *Server) newConnection(addr *net.UDPAddr, header protocol.Header, timeout int) *Connection {
	connection := &Connection{
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
	go connection.startTimer(timeout)

	return connection
}

func (s *Server) handleBufferParse(addr *net.UDPAddr, buffer []byte) {

	message, err := protocol.Unmarshal(buffer)
	if err != nil {
		return
	}

	if !message.Header.IsNil() {
		//Приводим к правильному формату данные
		message.Header.Login = strings.ToLower(message.Header.Login)
		message.Header.Domain = strings.ToUpper(message.Header.Domain)
		message.Header.Hostname = strings.ToUpper(message.Header.Hostname)

		//Подключаемся
		go s.receive(addr, message)
	}
}

func (s *Server) receive(addr *net.UDPAddr, message *protocol.UEP) {

	//Возвращаем подключение по имени компа
	//connection, ok := s.Connections.Get(message.Header.Hostname)
	v, ok := s.Connections.Load(message.Header.Hostname)
	if !ok {
		//Создаем и добавляем подключение
		//s.Connections.Add(message.Header.Hostname, newConnection(addr, message.Header))
		s.Connections.Store(message.Header.Hostname, NewConnection(addr, message.Header, s.Config.DisconnectTimeOut))
		return
	}
	//Обновляем данные по подключению
	connection := v.(*Connection)
	connection.update(addr, message.Header)

	if message.Body != nil {
		//Если есть данные с прицепом, то что то с ними делаем...
		go connection.Parse(*message.Body)
	}
}
/*
func stop() error {
	return srv.listener.Close()
}

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

func (s *Server) Send(hostname string, response *Response) (n int, err error) {

	//Проверяем на существование подключение
	//connection, ok := srv.Connections.Get(hostname)
	v, ok := s.Connections.Load(hostname)
	if !ok {
		return 0, errors.New(fmt.Sprintf("host: %s - подключение отсутствует!", hostname))
	}

	connection := v.(*Connection)

	//Сериализуем данные в json
	sendMessage, err := json.Marshal(response)
	if err != nil {
		return
	}

	//Запускаем отправку
	remoteAddr, err := net.ResolveUDPAddr(Udp4, connection.IpAddress.String())
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
/*
func send(hostname string, response *Response) (err error) {

	//Проверяем на существование подключение
	//connection, ok := srv.Connections.Get(hostname)
	v, ok := srv.Connections.Load(hostname)
	if !ok {
		return errors.New(fmt.Sprintf("host: %s - подключение отсутствует!", hostname))
	}

	connection := v.(*Connection)

	//Сериализуем данные в json
	sendMessage, err := json.Marshal(response)
	if err != nil {
		return
	}

	//Запускаем отправку
	remoteAddr, err := net.ResolveUDPAddr(Udp4, connection.IpAddress.String())
	if err != nil {
		return
		//logger.Save("server", "send", err.Error())
	}
	conn, err := net.DialUDP(Udp4, nil, remoteAddr)
	if err != nil {
		return
	}
	defer conn.Close()
	n, err := conn.Write(sendMessage)
	if err != nil {
		return
	}

	egologger.Info(send, hostname, fmt.Sprintf("%s(%d)", string(sendMessage), n))

	return
}*/
