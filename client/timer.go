package client

import (
	"github.com/egovorukhin/egotimer"
	"github.com/egovorukhin/egoudp/protocol"
	"time"
)

func (c *Client) startTimer(timeout int) {
	c.timer = egotimer.New(time.Duration(timeout)*time.Second, func(t time.Time) bool {
		if !c.Started.Get() {
			return true
		}
		if !c.Connected.Get() {
			c.packet.SetEvent(protocol.EventConnected)
			OnDisconnected(c.Handler, c)
			return true
		}
		c.Connected.Set(false)
		return false
	})
	go c.timer.Start()
}

func (c *Client) stopTimer() {
	if c.timer != nil {
		c.timer.Stop()
	}
}

func (c *Client) restartTimer() {
	if c.timer != nil {
		go c.timer.Restart()
	}
}
