package main

import (
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"github.com/egovorukhin/egoudp/server"
	"os"
	"strings"
)

func main() {
	config := server.Config{
		LocalPort:         5655,
		RemotePort:        5656,
		BufferSize:        4096,
		DisconnectTimeOut: 5,
	}
	udpserver := server.NewServer(config)
	udpserver.HandleConnected(OnConnected)
	udpserver.HandleDisconnected(OnDisconnected)
	udpserver.SetRoute("hi", func(c *server.Connection, resp protocol.IResponse, req protocol.Request) {
		resp.SetData(req.Data) //[]byte(`{"message": "Hello, world!"}`)
		_, err := c.Send(resp)
		if err != nil {
			fmt.Println(err)
		}
	})

	for {
		var input string
		_, err := fmt.Fscan(os.Stdin, &input)
		if err != nil {
			fmt.Println(err)
		}
		switch strings.ToLower(input) {
		case "start":
			err := udpserver.Start()
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("Сервер запущен")
			break
		case "stop":
			err := udpserver.Stop()
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("Сервер остановлен")
			break
		case "exit":
			os.Exit(0)
		}
	}
}

func OnConnected(c *server.Connection) error {
	fmt.Printf("Connected: %s(%s): %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
	return nil
}

func OnDisconnected(c *server.Connection) error {
	fmt.Printf("Disconnected: %s(%s) - %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
	return nil
}
