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
		Port:              5655,
		BufferSize:        4096,
		DisconnectTimeOut: 5,
		LogLevel:          0,
	}
	udpserver := server.NewServer(config)
	udpserver.HandleConnected(OnConnected)
	udpserver.HandleDisconnected(OnDisconnected)
	udpserver.SetRoute("hi", protocol.MethodNone, Hi)
	udpserver.SetRoute("winter", protocol.MethodGet, Winter)

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
		default:
			fmt.Println("Неизвестная команда")
		}
	}
}

func OnConnected(c *server.Connection) {
	fmt.Printf("Connected: %s(%s): %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
}

func OnDisconnected(c *server.Connection) {
	fmt.Printf("Disconnected: %s(%s) - %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
}

func Hi(c *server.Connection, resp protocol.IResponse, req protocol.Request) {
	resp.SetData(req.Data)
	fmt.Println(string(req.Data))
	_, err := c.Send(resp)
	if err != nil {
		fmt.Println(err)
	}
}

func Winter(c *server.Connection, resp protocol.IResponse, req protocol.Request) {
	//JSON
	data := `["Декабрь", "Январь", "Февраль"]`
	resp = resp.SetData([]byte(data)).SetContentType("json")
	_, err := c.Send(resp)
	if err != nil {
		fmt.Println(err)
	}
}
