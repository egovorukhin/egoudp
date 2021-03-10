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

//События
type HandleClient func(c *Client)

type Client struct {
	Config
	connection *net.UDPConn
	packet     *protocol.Packet
	*log.Logger
	queue                 sync.Map
	timer                 *egotimer.Timer
	handleStart           HandleClient
	handleStop            HandleClient
	handleConnected       HandleClient
	handleDisconnected    HandleClient
	handleCheckConnection HandleClient
	sync.Mutex
	Connected bool
	Started   bool
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
	HandleStart(handler HandleClient)
	HandleStop(handler HandleClient)
	HandleConnected(handler HandleClient)
	HandleDisconnected(handler HandleClient)
	HandleCheckConnection(handler HandleClient)
}

const udp = "udp"

func New(config Config) IClient {
	return &Client{
		Config: config,
		Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
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

	c.Started = true

	//отправка пакетов
	go c.send()
	//прием пакетов
	go c.receive()

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
			c.Started = false
			break
		}

		time.Sleep(time.Second * 1)
	}
}

//Прием данных
func (c *Client) receive() {

	buffer := make([]byte, c.BufferSize)

	for {

		if !c.Started {
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
		c.Lock()
		c.Connected = true
		c.Unlock()
		if c.handleConnected != nil {
			go c.handleConnected(c)
		}
		if resp.Data != nil {
			timeout, err := strconv.Atoi(string(resp.Data))
			if err != nil {
				return err
			}
			go c.startTimer(timeout)
		}
		break
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//событие отключения клиента
		c.Lock()
		c.Connected = false
		c.Unlock()

		c.timer.Stop()

		if c.handleDisconnected != nil {
			go c.handleDisconnected(c)
		}

		break
	case protocol.EventCheckConnection:
		//событие проверки активности сервера
		c.Lock()
		c.Connected = true
		c.Unlock()

		go c.timer.Restart()

		if c.handleCheckConnection != nil {
			go c.handleCheckConnection(c)
		}
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

	if !c.Connected {
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

func (c *Client) startTimer(timeout int) {
	c.timer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		if !c.Started {
			return true
		}
		c.Lock()
		defer c.Unlock()
		if !c.Connected {
			c.packet.SetEvent(protocol.EventConnected)
			if c.handleDisconnected != nil {
				go c.handleDisconnected(c)
			}
			return true
		}
		c.Connected = false
		return false
	})
	defer c.timer.Stop()
	c.timer.Start()
}

func (c *Client) id() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}

func (c *Client) HandleConnected(handler HandleClient) {
	c.handleConnected = handler
}

func (c *Client) HandleDisconnected(handler HandleClient) {
	c.handleDisconnected = handler
}

func (c *Client) HandleCheckConnection(handler HandleClient) {
	c.handleCheckConnection = handler
}

func (c *Client) HandleStart(handler HandleClient) {
	c.handleStart = handler
}

func (c *Client) HandleStop(handler HandleClient) {
	c.handleStop = handler
}

func (c *Client) Stop() {
	if c.handleStop != nil {
		c.handleStop(c)
	}
	c.Lock()
	c.Connected = false
	c.Unlock()
	c.packet.SetEvent(protocol.EventDisconnect)
}
