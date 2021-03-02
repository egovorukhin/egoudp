package main

import (
	"fmt"
	"github.com/egovorukhin/egoudp/client"
	"github.com/egovorukhin/egoudp/protocol"
	"os"
	"strings"
	"time"
)

func main() {
	config := client.Config{
		RemoteHost: "10.28.0.73",
		//LocalPort:  5656,
		RemotePort: 5655,
		BufferSize: 4096,
		TimeOut:    30,
	}
	udpclient := client.NewClient(config)
	udpclient.HandleConnected(OnConnected)
	udpclient.HandleDisconnected(OnDisconnected)
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	for {
		var input string
		_, err := fmt.Fscan(os.Stdin, &input)
		if err != nil {
			fmt.Println(err)
		}
		switch strings.ToLower(input) {
		case "start":
			err = udpclient.Start(hostname, "login", "domain.com", "1.0.0")
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("Клиент запущен")
			break
		case "send":
			req := protocol.NewRequest("hi", protocol.MethodNone).
				SetData("json", []byte(`{"message": "Hello, World!"}`))
			fmt.Println("Сообщение отправлено")
			resp := udpclient.Send(req)
			fmt.Println(resp)
		case "stop":
			udpclient.Stop()
			fmt.Println("Клиент остановлен")
			break
		case "exit":
			os.Exit(0)
		}
	}
}

func OnConnected(c *client.Client) {
	fmt.Printf("Connected: %s\n", time.Now().Format("15:04:05"))
}

func OnDisconnected(c *client.Client) {
	fmt.Printf("Disconnected: %s\n", time.Now().Format("15:04:05"))
}
