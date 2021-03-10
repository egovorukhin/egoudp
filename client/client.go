package client

import (
	"errors"
	"fmt"
	"github.com/egovorukhin/egotimer"
	"github.com/egovorukhin/egoudp/protocol"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type QItem struct {
	Request  *protocol.Request
	Response *protocol.Response
	Sent     bool
	Received bool
}

type Client struct {
	Config
	connection *net.UDPConn
	packet     *protocol.Packet
	queue      sync.Map
	timer      *egotimer.Timer
	Connected  Connected
	Started    Started
	*log.Logger
	Handler *Handler
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

type Connected struct {
	sync.Mutex
	value bool
}

func (c *Connected) Set(b bool) {
	c.Lock()
	c.value = b
	c.Unlock()
}

func (c *Connected) Get() bool {
	c.Lock()
	defer c.Unlock()
	return c.value
}

type Config struct {
	Host       string
	Port       int
	BufferSize int
	Timeout    int
	LogLevel   LogLevel
}

type LogLevel int

const (
	LogLevelLow LogLevel = iota
	LogLevelHigh
)

type IClient interface {
	Start(hostname, login, domain, version string) error
	Stop()
	SetLogger(out io.Writer, prefix string, flag int)
	Send(req *protocol.Request) (*protocol.Response, error)
	OnStart(handler HandleClient)
	OnStop(handler HandleClient)
	OnConnected(handler HandleClient)
	OnDisconnected(handler HandleClient)
	OnCheckConnection(handler HandleClient)
}

const udp = "udp"

func New(config Config) IClient {
	return &Client{
		Config:  config,
		Logger:  log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Handler: new(Handler),
	}
}

func (c *Client) SetLogger(out io.Writer, prefix string, flag int) {
	c.Logger = log.New(out, prefix, flag)
}

func (c *Client) Start(hostname, login, domain, version string) error {

	remoteAdder, err := net.ResolveUDPAddr(udp, fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		return err
	}

	c.connection, err = net.DialUDP(udp, nil, remoteAdder)
	if err != nil {
		return err
	}

	c.packet = protocol.New(hostname, login, domain, version)
	c.packet.Event = protocol.EventConnected

	c.Started.value = true

	//отправка пакетов
	go c.send()
	//прием пакетов
	go c.receive()

	OnStart(c.Handler, c)

	return nil
}

//Отправка данных.
func (c *Client) send() {

	defer c.connection.Close()

	for {

		//Пробегаемся по очереди и заполняем Request,
		//false - выходим из цикла, true - продолжаем крутить
		c.queue.Range(func(key, value interface{}) bool {
			item := value.(*QItem)
			if !item.Sent {
				c.packet.Request = item.Request
				item.Sent = true
				return false
			}
			return true
		})

		//Пишем данные в порт
		n, err := c.connection.Write(c.packet.Marshal())
		if err != nil {
			c.Println(err)
		}

		if c.LogLevel == LogLevelHigh {
			c.Printf("%s(%d)", c.packet.String(), n)
		}

		//Очищаем Request
		c.packet.Request = nil

		if c.packet.GetEvent() == protocol.EventDisconnect {
			break
		}

		time.Sleep(time.Second * 1)
	}
}

//Прием данных
func (c *Client) receive() {

	buffer := make([]byte, c.BufferSize)

	for {

		if !c.Started.Get() {
			break
		}

		n, _, err := c.connection.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		//Передаем данные и разбираем их
		go func() {
			err = c.handleBufferParse(buffer[:n])
			if err != nil {
				c.Println(err)
			}
		}()
	}
}

//Функция парсинга входных данных.
func (c *Client) handleBufferParse(buffer []byte) error {

	resp := new(protocol.Response)
	err := resp.Unmarshal(buffer)
	if err != nil {
		return err
	}

	//Проверяем события
	switch resp.Event {
	//Отправляем команду о подключении клиенту
	case protocol.EventConnected:
		//событие подключения клиента
		c.Connected.Set(true)
		OnConnected(c.Handler, c)
		if resp.Data != nil {
			timeout, err := strconv.Atoi(string(resp.Data))
			if err != nil {
				return err
			}
			c.startTimer(timeout)
		}
		break
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//событие отключения клиента
		c.Connected.Set(false)
		c.stopTimer()
		OnDisconnected(c.Handler, c)
		return nil
	case protocol.EventCheckConnection:
		//событие проверки активности сервера
		c.Connected.Set(true)
		c.restartTimer()
		OnCheckConnection(c.Handler, c)
		break
	}

	if c.packet.GetEvent() != protocol.EventNone {
		c.packet.SetEvent(protocol.EventNone)
	}

	go func() {
		v, ok := c.queue.Load(resp.Id)
		if ok {
			v.(*QItem).Response = resp
			v.(*QItem).Received = true
		}
	}()

	return nil
}

//Отправка запроса на сервер. Добавляем в очередь запрос
//и запускаем wait функцию
func (c *Client) Send(req *protocol.Request) (*protocol.Response, error) {
	req.Id = c.id()
	c.queue.Store(req.Id, &QItem{
		Request:  req,
		Sent:     false,
		Received: false,
	})

	//Ждем ответа
	resp := make(chan *protocol.Response)
	err := make(chan error)
	go c.wait(req.Id, resp, err)
	return <-resp, <-err
}

//Ждем ответ от сервера на наш запрос.
//Если в течении timeout не придет ответ, то возвращаем nil
func (c *Client) wait(id string, resp chan *protocol.Response, err chan error) {

	if !c.Connected.Get() {
		resp <- nil
		err <- errors.New("Клиент не подключен к серверу")
	}

	count := 0
	timer := egotimer.New(1*time.Second, func(t time.Time) bool {
		if count == c.Timeout {
			return true
		}
		count++
		return false
	})
	defer timer.Stop()
	go timer.Start()

	received := false
	for count < c.Timeout && !received {
		c.queue.Range(func(key, value interface{}) bool {
			item := value.(*QItem)
			if key == id && item.Received {
				resp <- item.Response
				err <- nil
				received = true
			}
			return !received
		})
	}
	c.queue.Delete(id)
	if received {
		return
	}
	resp <- nil
	err <- errors.New("Вышло время ожидания запроса")
}

func (c *Client) id() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}

func (c *Client) OnConnected(handler HandleClient) {
	c.Handler.OnConnected = handler
}

func (c *Client) OnDisconnected(handler HandleClient) {
	c.Handler.OnDisconnected = handler
}

func (c *Client) OnCheckConnection(handler HandleClient) {
	c.Handler.OnCheckConnection = handler
}

func (c *Client) OnStart(handler HandleClient) {
	c.Handler.OnStart = handler
}

func (c *Client) OnStop(handler HandleClient) {
	c.Handler.OnStop = handler
}

func (c *Client) Stop() {
	OnStop(c.Handler, c)
	c.timer.Stop()
	c.Started.Set(false)
	c.Connected.Set(false)
	c.packet.SetEvent(protocol.EventDisconnect)
}
