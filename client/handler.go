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

func (h *Handler) HandleStart(c *Client) {
	if h.OnStart != nil {
		go h.OnStart(c)
	}
}

func (h *Handler) HandleStop(c *Client) {
	if h.OnStop != nil {
		go h.OnStop(c)
	}
}

func (h *Handler) HandleConnected(c *Client) {
	if h.OnConnected != nil {
		go h.OnConnected(c)
	}
}

func (h *Handler) HandleDisconnected(c *Client) {
	if h.OnDisconnected != nil {
		go h.OnDisconnected(c)
	}
}

func (h *Handler) HandleCheckConnection(c *Client) {
	if h.OnCheckConnection != nil {
		go h.OnCheckConnection(c)
	}
}

type IHandler interface {
	HandleStart(c *Client)
	HandleStop(c *Client)
	HandleConnected(c *Client)
	HandleDisconnected(c *Client)
	HandleCheckConnection(c *Client)
}

func OnStart(handler IHandler, c *Client) {
	handler.HandleStart(c)
}

func OnStop(handler IHandler, c *Client) {
	handler.HandleStop(c)
}

func OnConnected(handler IHandler, c *Client) {
	handler.HandleConnected(c)
}

func OnDisconnected(handler IHandler, c *Client) {
	handler.HandleDisconnected(c)
}

func OnCheckConnection(handler IHandler, c *Client) {
	handler.HandleCheckConnection(c)
}
