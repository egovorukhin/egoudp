package main

import (
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
	"github.com/egovorukhin/egoudp/server"
	"os"
	"strings"
	"time"
)

type Events int

const (
	EventNotify Events = 3
)

func main() {
	config := server.Config{
		Port:                   5655,
		BufferSize:             256,
		DisconnectTimeout:      5,
		CheckConnectionTimeout: 30,
		LogLevel:               0,
	}
	srv := server.New(config)
	srv.OnConnected(OnConnected)
	srv.OnDisconnected(OnDisconnected)
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
			err = srv.Stop()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Сервер остановлен")
			break
		case "get_connections":
			fmt.Println(srv.GetConnections())
			break
		case "get_routes":
			fmt.Println(srv.GetRoutes())
			break
		case "nf":
			resp := &protocol.Response{
				StatusCode:  protocol.StatusCodeOK,
				Event:       int(EventNotify),
				ContentType: "",
				Data:        protocol.ToRunes("Как жизнь?"),
			}
			srv.Send("gb1-dit-1-16146", resp)
			break
		case "exit":
			os.Exit(0)
		default:
			fmt.Println("Неизвестная команда")
		}
	}
}

func OnConnected(c *server.Connection) {
	fmt.Printf("OnConnected: %s(%s): %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
}

func OnDisconnected(c *server.Connection) {
	fmt.Printf("OnDisconnected: %s(%s) - %s\n", c.Hostname, c.IpAddress.String(), c.DisconnectTime.Format("15:04:05"))
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
	resp = resp.SetData(protocol.StatusCodeOK, []rune(data)).SetContentType("json")
	_, err := c.Send(resp)
	if err != nil {
		fmt.Println(err)
	}
}
