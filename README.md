## **Описание**

**EgoUDP** - модуль для проектов на GO, который включает в себя сервер и клиент.

## **Установка**

Для установки модуля доcтаточно использовать `go get` команду:
```
go get -u github.com/egovorukhin/egoudp
```
## **Быстрый старт**

### Сервер
* **Конфигурация**

```golang
  import "github.com/egovorukhin/egoudp/server"

  config := server.Config{
          Port:              5655,
          BufferSize:        4096,
          DisconnectTimeOut: 5,
          LogLevel:          0,
      }
  srv := server.New(config)
```
Заполняем конфигупацию для сервера. `Port` - порт по котороому сервер будет принимать данные. `BufferSize` - размер входного буфера. Когда перестают приходить пакеты от клиента, то подключение через `DisconnectTimeOut` секунд удаляется из памяти. `LogLevel` - уровень логиролвания.

* **События**
```golang
  srv.HandleConnected(OnConnected)
  srv.HandleDisconnected(OnDisconnected)
```
```golang
  func OnConnected(c *server.Connection) {
      fmt.Printf("Connected: %s(%s): %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
  }

  func OnDisconnected(c *server.Connection) {
      fmt.Printf("Disconnected: %s(%s) - %s\n", c.Hostname, c.IpAddress.String(), c.ConnectTime.Format("15:04:05"))
  }
```
Можем определить функции для событий подключения/отключения клиентов, главное соблюсти вид функций.

* **Маршруты**
```
  srv.SetRoute("hi", protocol.MethodNone, Hi)
  srv.SetRoute("winter", protocol.MethodGet, Winter)
```
Устанавливаем маршруты по аналогии с http протоколом. `path` - путь для определения маршрутв. `method` - метод для определенного маршрута. `handler` - функция которая выполнится при запросе от клиента по определенному пути маршрута.
```
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
```
Определяем функции для маршрутов вида `func(c *Connection, resp protocol.IResponse, req protocol.Request)`. `c *Connection` - передается подключение, которое хранит всю информация об этом подключении. `resp protocol.IResponse` - интерфейс который мы используем для заполнения ответа на запрос. `req protocol.Request` - запрос от клиента.

* **Логирование**
```
  f, _ := os.Open(path)
  srv.SetLogger(f, "", log.Ldate|log.Ltime)
```
Можно переопределить `Writer` для `log.Logger`, по умолчанию вывод будет происходить на `os.Stdout`.

* **Запуск**

```
  _ = srv.Start()
```
`Start()` - запускает сервер, возвращает ошибку `error`

* **Остановка**
```
  _ = srv.Stop()
```
`Stop()` - остановка сервера, возвращает ошибку `error`

---

### Клиент

* **Конфигурация**
```
  import "github.com/egovorukhin/egoudp/client"

  config := client.Config{
          Host:       "localhost",
          Port:       5655,
          BufferSize: 4096,
          TimeOut:    30,
          LogLevel:   0,
      }
  clt := client.New(config)
```
Заполняем конфигупацию для сервера. `Host` - имя сервера.`Port` - порт по котороому клиент будет отправлять данные. `BufferSize` - размер входного буфера. `TimeOut` - тайаут ответа, т.е. ответ должен прийти в течении этого времени. `LogLevel` - уровень логиролвания.

* **События**
```golang
  clt.HandleConnected(OnConnected)
  clt.HandleDisconnected(OnDisconnected)
```
```golang
  func OnConnected(c *client.Client) {
      fmt.Printf("Connected: %s\n", time.Now().Format("15:04:05"))
  }

  func OnDisconnected(c *client.Client) {
      fmt.Printf("Disconnected: %s\n", time.Now().Format("15:04:05"))
  }
```
Можем определить функции для событий подключения/отключения клиентов, главное соблюсти вид функций.

* **Запуск**
```
  hostname, _ := os.Hostname()
  _ = clt.Start(hostname, "login", "domain.com", "1.0.0")
```
`Start` - запуск клиента. Необходимо передать обязательные аргументы. `hostname` - имя машины где стоит клиент. `login` - учетная запись под которой запущен клиент. `domain` - домен под которой запущен клиент. `version` - версия вашего  разрабатываемого приложения. Возвращает `error`.

* **Запросы**
```
  req := protocol.NewRequest("hi", protocol.MethodNone).SetData("json", []byte(`{"message": "Hello, World!"}`))
  resp, _ := c.Send(req)
  fmt.Println(resp.Data)
```
```
  var w []string
  req := protocol.NewRequest("winter", protocol.MethodGet)
  resp, _ := c.Send(req)
  if resp.ContentType == "json" {
  	_ = json.Unmarshal(resp.Data, &w)
  }
  fmt.Println(w)

```
`NewRequest` - инициализация запроса. `SetData` - передаем вид данных и сами данные в `[]byte`. `Send(req *Request)` - отправка запроса на сервер, возвращает `*Response, error`.

* **Логирование**
```
  f, _ := os.Open(path)
  clt.SetLogger(f, "", log.Ldate|log.Ltime)
```
Можно переопределить `Writer` для `log.Logger`, по умолчанию вывод будет происходить на `os.Stdout`.

* **Остановка**
```
  _ = clt.Stop()
```
`Stop()` - остановка клиента, возвращает ошибку `error`.