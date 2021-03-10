package server

//События сервера
type HandleServer func(s *Server)

//События подключений
type HandleConnection func(c *Connection)

type Handler struct {
	OnStart        HandleServer
	OnStop         HandleServer
	OnConnected    HandleConnection
	OnDisconnected HandleConnection
}

func (h *Handler) HandleStart(s *Server) {
	if h.OnStart != nil {
		go h.OnStart(s)
	}
}

func (h *Handler) HandleStop(s *Server) {
	if h.OnStop != nil {
		go h.OnStop(s)
	}
}

func (h *Handler) HandleConnected(c *Connection) {
	if h.OnConnected != nil {
		go h.OnConnected(c)
	}
}

func (h *Handler) HandleDisconnected(c *Connection) {
	if h.OnDisconnected != nil {
		go h.OnDisconnected(c)
	}
}

type IHandler interface {
	HandleStart(s *Server)
	HandleStop(s *Server)
	HandleConnected(c *Connection)
	HandleDisconnected(c *Connection)
}

func OnStart(handler IHandler, s *Server) {
	handler.HandleStart(s)
}

func OnStop(handler IHandler, s *Server) {
	handler.HandleStop(s)
}

func OnConnected(handler IHandler, c *Connection) {
	handler.HandleConnected(c)
}

func OnDisconnected(handler IHandler, c *Connection) {
	handler.HandleDisconnected(c)
}
