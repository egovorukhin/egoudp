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
	Config             Config
	sender             *net.UDPConn
	listener           *net.UDPConn
	Started            bool
	Connected          bool
	packet             *protocol.Packet
	queue              sync.Map
	handleConnected    HandleConnected
	handleDisconnected HandleDisconnected
}

type Config struct {
	LocalPort  int
	RemotePort int
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

const Udp4 = "udp4"

func NewClient(config Config) IClient {
	return &Client{
		Config: config,
	}
}

func (c *Client) Start(hostname, login, domain, version string) error {

	remoteAdder, err := net.ResolveUDPAddr(Udp4, ":"+strconv.Itoa(c.Config.RemotePort))
	if err != nil {
		return err
	}
	/*
		localAddr, err := net.ResolveUDPAddr(Udp4, ":"+strconv.Itoa(c.Config.LocalPort))
		if err != nil {
			return err
		}*/

	c.sender, err = net.DialUDP(Udp4, nil, remoteAdder)
	if err != nil {
		return err
	}

	c.listener, err = net.ListenUDP(Udp4, nil)
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
				c.packet.Request = item.Request
				item.sended = true
			}
			return true
		})

		_, err := c.sender.Write(c.packet.Marshal())
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(time.Second * 1)
	}

}

func (c *Client) receive() {

	for {

		if !c.Started {
			break
		}

		buffer := make([]byte, c.Config.BufferSize)

		n, addr, err := c.listener.ReadFromUDP(buffer)
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
		go c.handleConnected(c)
		return
	//Команда на отключение клиента
	case protocol.EventDisconnect:
		//событие отключения клиента
		c.Connected = false
		c.handleDisconnected(c)
		err = c.sender.Close()
		if err != nil {
			fmt.Println(err)
		}
		err = c.listener.Close()
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	go func(resp *protocol.Response) {
		c.queue.Range(func(key, value interface{}) bool {
			if key == resp.Id {
				value.(*Item).Response = resp
				value.(*Item).received = true
				return true
			}
			return false
		})
	}(resp)
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
	//ticker := time.NewTicker(time.Duration(c.Config.TimeOut) * time.Second)
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()
	//go func() {
	for range timer.C {
		c.queue.Range(func(key, value interface{}) bool {
			if key == id && value.(*Item).received {

			}
			return false
		})

		return
	}
	resp <- nil
	//}()
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
	c.Started = false
	c.packet.Header.Event = protocol.EventDisconnect
}
