package client

//События клиента
type HandleClient func(c *Client)

type Handler struct {
	OnStart           HandleClient
	OnStop            HandleClient
	OnConnected       HandleClient
	OnDisconnected    HandleClient
	OnCheckConnection HandleClient
}

type IHandler interface {
	HandleStart(s *Server)
	HandleStop(s *Server)
	HandleConnected(c *Connection)
	HandleDisconnected(c *Connection)
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
