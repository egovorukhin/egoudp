package client

import (
	"errors"
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"github.com/google/uuid"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type QItem struct {
	Request  *protocol.Request
	Response *protocol.Response
	sended   bool
	received bool
}

//События
type HandleConnected func(c *Client)
type HandleDisconnected func(c *Client)

type Client struct {
	Config
	*net.UDPConn
	*log.Logger
	Started   bool
	Connected bool
	sync.RWMutex
	packet             *protocol.Packet
	queue              sync.Map
	handleConnected    HandleConnected
	handleDisconnected HandleDisconnected
}

type Config struct {
	Host       string
	Port       int
	BufferSize int
	TimeOut    int
	LogLevel   LogLevel
}

type LogLevel int

const (
	LogLevelLow LogLevel = iota
	LogLevelHigh
)

type IClient interface {
	Start(hostname, login, domain, version string) error
	Stop() error
	SetLogger(out io.Writer, prefix string, flag int)
	Send(req *protocol.Request) (*protocol.Response, error)
	HandleConnected(handler HandleConnected)
	HandleDisconnected(handler HandleDisconnected)
}

const udp = "udp"

func NewClient(config Config) IClient {
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

	c.UDPConn, err = net.DialUDP(udp, nil, remoteAdder)
	if err != nil {
		return err
	}

	c.packet = protocol.NewUEP(hostname, login, domain, version)
	c.packet.Header.Event = protocol.EventConnected

	c.Started = true

	//отправка пакетов
	go c.send()
	//прием пакетов
	go c.receive()

	return nil
}

func (c *Client) send() {

	for {

		if !c.Started {
			break
		}

		c.queue.Range(func(key, value interface{}) bool {
			item := value.(*QItem)
			if !item.sended {
				c.Lock()
				c.packet.Request = item.Request
				c.Unlock()
				item.sended = true
			}

			return true
		})

		n, err := c.Write(c.packet.Marshal())
		if err != nil {
			c.Println(err)
		}

		if c.LogLevel == LogLevelHigh {
			c.Printf("%s(%d)", c.packet.String(), n)
		}

		c.packet.Request = nil

		time.Sleep(time.Second * 1)
	}
}

func (c *Client) receive() {

	buffer := make([]byte, c.BufferSize)

	for {

		if !c.Started {
			break
		}

		n, addr, err := c.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		//Передаем данные и разбираем их
		go c.handleBufferParse(addr, buffer[:n])
	}
}

func (c *Client) handleBufferParse(addr *net.UDPAddr, buffer []byte) {

	resp := new(protocol.Response)
	err := resp.Unmarshal(buffer)
	if err != nil {
		return
	}

	//Проверяем события
	switch resp.Event {
	//Отправляем команду о подключении клиенту
	case protocol.EventConnected:
		//событие подключения клиента
		c.Connected = true
		c.Lock()
		c.packet.Header.Event = protocol.EventNone
		c.Unlock()
		go c.handleConnected(c)
		return
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//событие отключения клиента
		c.Connected = false
		c.Lock()
		c.packet.Header.Event = protocol.EventNone
		c.Unlock()
		return
	}

	go func() {
		v, ok := c.queue.Load(resp.Id)
		if ok {
			v.(*QItem).Response = resp
			v.(*QItem).received = true
		}
	}()
}

func (c *Client) Send(req *protocol.Request) (*protocol.Response, error) {
	req.Id = c.id()
	c.queue.Store(req.Id, &QItem{
		Request: req,
	})

	//Ждем ответа
	resp := make(chan *protocol.Response)
	err := make(chan error)
	go c.wait(req.Id, resp, err)
	return <-resp, <-err
}

func (c *Client) wait(id string, resp chan *protocol.Response, err chan error) {

	if !c.Connected {
		resp <- nil
		err <- errors.New("Клиент не подключен к серверу")
	}

	i := 0
	go func() {
		for {
			if i == c.TimeOut {
				return
			}
			time.Sleep(1 * time.Second)
			i++
		}
	}()

	received := false
	for i < c.TimeOut {
		c.queue.Range(func(key, value interface{}) bool {
			item := value.(*QItem)
			if key == id && item.received {
				resp <- item.Response
				err <- nil
				received = true
			}
			return received
		})
		if received {
			c.queue.Delete(id)
			return
		}
	}
	resp <- nil
	err <- errors.New("Вышло время ожидания запроса")
}

func (c *Client) id() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}

func (c *Client) HandleConnected(handler HandleConnected) {
	c.handleConnected = handler
}

func (c *Client) HandleDisconnected(handler HandleDisconnected) {
	c.handleDisconnected = handler
}

func (c *Client) Stop() error {
	c.handleDisconnected(c)
	err := c.Close()
	if err != nil {
		return err
	}
	c.Started = false
	c.Lock()
	c.packet.Header.Event = protocol.EventDisconnect
	c.Unlock()

	return nil
}
