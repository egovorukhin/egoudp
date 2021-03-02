package client

import (
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"github.com/google/uuid"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Item struct {
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
	Started   bool
	Connected bool
	sync.RWMutex
	packet             *protocol.Packet
	queue              sync.Map
	handleConnected    HandleConnected
	handleDisconnected HandleDisconnected
}

type Config struct {
	RemoteHost string
	RemotePort int
	LocalPort  int
	BufferSize int
	TimeOut    int
}

type IClient interface {
	Start(hostname, login, domain, version string) error
	Stop()
	Send(req *protocol.Request) *protocol.Response
	HandleConnected(handler HandleConnected)
	HandleDisconnected(handler HandleDisconnected)
}

const udp = "udp"

func NewClient(config Config) IClient {
	return &Client{
		Config: config,
	}
}

func (c *Client) Start(hostname, login, domain, version string) error {

	remoteAdder, err := net.ResolveUDPAddr(udp, fmt.Sprintf("%s:%d", c.RemoteHost, c.RemotePort))
	if err != nil {
		return err
	}

	localAddr, err := net.ResolveUDPAddr(udp, ":"+strconv.Itoa(c.LocalPort))
	if err != nil {
		return err
	}

	c.UDPConn, err = net.DialUDP(udp, localAddr, remoteAdder)
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
			item := value.(*Item)
			if !item.sended {
				c.Lock()
				c.packet.Request = item.Request
				c.Unlock()
				item.sended = true
			}

			return true
		})

		_, err := c.Write(c.packet.Marshal())
		if err != nil {
			fmt.Println(err)
		}

		c.packet.Request = nil

		time.Sleep(time.Second * 1)
	}

}

func (c *Client) receive() {

	for {

		if !c.Started {
			break
		}

		buffer := make([]byte, c.BufferSize)

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
			v.(*Item).Response = resp
			v.(*Item).received = true
		}
	}()
}

func (c *Client) Send(req *protocol.Request) *protocol.Response {
	c.queue.Store(c.id(), &Item{
		Request: req,
	})

	//Ждем ответа
	resp := make(chan *protocol.Response)
	go c.wait(req.Id, resp)
	return <-resp
}

func (c *Client) wait(id string, resp chan *protocol.Response) {

	resp <- nil

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
			item := value.(*Item)
			if key == id && item.received {
				resp <- item.Response
				received = true
			}
			return received
		})
		if received {
			return
		}
	}
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

func (c *Client) Stop() {
	c.handleDisconnected(c)
	err := c.Close()
	if err != nil {
		fmt.Println(err)
	}
	/*err = c.listener.Close()
	if err != nil {
		fmt.Println(err)
	}*/
	c.Started = false
	c.Lock()
	c.packet.Header.Event = protocol.EventDisconnect
	c.Unlock()
}
