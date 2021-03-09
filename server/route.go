package server

import (
	"fmt"
	"github.com/egovorukhin/egoudp/protocol"
)

//Функция которая вызывается при событии получения определённого маршоута
type FuncHandler func(c *Connection, resp protocol.IResponse, req protocol.Request)

type Route struct {
	Path    string
	Method  protocol.Methods
	Handler FuncHandler
}

func (r *Route) String() string {
	return fmt.Sprintf("path: %s, method: %s", r.Path, r.Method.String())
}
