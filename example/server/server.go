package main

import (
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"github.com/egovorukhin/egoudp/server"
	"os"
	"strings"
	"time"
)

func main() {
	config := server.Config{
		Port:                   5655,
		BufferSize:             256,
		DisconnectTimeout:      5,
		CheckConnectionTimeout: 3,
		LogLevel:               0,
	}
	srv := server.New(config)
	srv.HandleConnected(OnConnected)
	srv.HandleDisconnected(OnDisconnected)
	srv.SetRoute("hi", protocol.MethodNone, Hi)
	srv.SetRoute("winter", protocol.MethodGet, Winter)

	for {
		var input string
		_, err := fmt.Fscan(os.Stdin, &input)
		if err != nil {
			fmt.Println(err)
		}
		switch strings.ToLower(input) {
		case "start":
			err := srv.Start()
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("Сервер запущен")
			break
		case "stop":
			srv.Stop()
			fmt.Println("Сервер остановлен")
			break
		case "get_connections":
			fmt.Println(srv.GetConnections())
			break
		case "get_routes":
			fmt.Println(srv.GetRoutes())
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
	resp.SetData(protocol.StatusCodeOK, req.Data)
	fmt.Println(string(req.Data))
	time.Sleep(10 * time.Second)
	_, err := c.Send(resp)
	if err != nil {
		fmt.Println(err)
	}
}

func Winter(c *server.Connection, resp protocol.IResponse, req protocol.Request) {
	//JSON
	data := `["Декабрь", "Январь", "Февраль"]`
	resp = resp.SetData(protocol.StatusCodeOK, []byte(data)).SetContentType("json")
	_, err := c.Send(resp)
	if err != nil {
		fmt.Println(err)
	}
}
