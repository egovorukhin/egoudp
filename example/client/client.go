package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/egovorukhin/egoudp/client"
	"github.com/egovorukhin/egoudp/protocol"
	"os"
	"strings"
	"time"
)

func main() {
	config := client.Config{
		Host:       "localhost",
		Port:       5655,
		BufferSize: 4096,
		Timeout:    30,
		LogLevel:   0,
	}
	clt := client.New(config)
	clt.OnConnected(OnConnected)
	clt.OnDisconnected(OnDisconnected)
	clt.OnCheckConnection(OnCheckConnection)
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
			err = clt.Start(hostname, "login", "domain.com", "1.0.0")
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("Клиент запущен")
			break
		case "hi":
			go func() {
				b, err := Hi(clt)
				if err != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(string(b))
			}()
			go func() {
				b, err := Hi(clt)
				if err != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(string(b))
			}()
			/*b, err := Hi(clt)
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println(string(b))*/
			break
		case "winter":
			var w []string
			err := Winter(clt, &w)
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println(w)
			break
		case "stop":
			clt.Stop()
			fmt.Println("Клиент остановлен")
			break
		case "exit":
			os.Exit(0)
		default:
			fmt.Println("Неизвестная команда")
		}
	}
}

func OnConnected(c *client.Client) {
	fmt.Printf("OnConnected: %s\n", time.Now().Format("15:04:05"))
}

func OnDisconnected(c *client.Client) {
	fmt.Printf("OnDisconnected: %s\n", time.Now().Format("15:04:05"))
}

func OnCheckConnection(c *client.Client) {
	fmt.Printf("CheckConnection: %s\n", time.Now().Format("15:04:05"))
}

func Hi(c client.IClient) ([]rune, error) {
	req := protocol.NewRequest("hi", protocol.MethodNone).
		SetData("json", []rune(`{"message": "Hello, World!"}`))
	resp, err := c.Send(req)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func Winter(c client.IClient, v interface{}) error {
	req := protocol.NewRequest("winter", protocol.MethodGet)
	resp, err := c.Send(req)
	if err != nil {
		return err
	}
	switch resp.ContentType {
	case "json":
		err = json.Unmarshal(resp.Data.ToByte(), v)
		if err != nil {
			return err
		}
		break
	case "xml":
		err = xml.Unmarshal(resp.Data.ToByte(), v)
		if err != nil {
			return err
		}
	}

	return nil
}
