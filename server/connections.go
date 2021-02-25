package server

import "sync"

type Connections struct {
	m map[string]Connection
	sync.RWMutex
}

func NewConnections() *Connections {
	return &Connections{
		m: make(map[string]Connection),
	}
}

func (c *Connections) Add(hostname string, connection Connection) {
	c.Lock()
	defer c.Unlock()
	c.m[hostname] = connection
}

func (c *Connections) Get(hostname string) (Connection, bool) {
	c.RLock()
	defer c.RUnlock()
	connection, ok := c.m[hostname]
	return connection, ok
}

/*
func (c *Connections) Delete(hostname string)  {
	_, ok := c.Get(hostname)
	if ok {
		c.Lock()
		delete(c, hostname)
		c.Unlock()
	}
}*/
