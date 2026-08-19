package tui

import (
	"ps4rpc/internal/daemon"
	"ps4rpc/internal/ipc"

	tea "github.com/charmbracelet/bubbletea"
)

type conn struct {
	role   string
	client *ipc.Client
	logCh  chan string
}

func newConn(role string, logCh chan string) *conn {
	c := &conn{role: role, logCh: logCh}
	c.dial()
	return c
}

func (c *conn) dial() bool {
	if c.client != nil {
		return true
	}
	client, err := ipc.Dial(c.role)
	if err != nil {
		return false
	}
	c.client = client

	var history []string
	if err := client.Call(ipc.MethodLogs, nil, &history); err == nil {
		for _, line := range history {
			c.push(line)
		}
	}
	if err := client.Call(ipc.MethodSubscribe, nil, nil); err != nil {
		c.drop()
		return false
	}
	go func(events <-chan ipc.Event) {
		for ev := range events {
			if ev.Name == ipc.EventLog {
				c.push(ev.Str())
			}
		}
	}(client.Events())
	return true
}

func (c *conn) push(line string) {
	select {
	case c.logCh <- line:
	default:
	}
}

func (c *conn) drop() {
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
}

func (c *conn) call(method string, out any) error {
	if c.client == nil {
		return ipc.ErrClosed
	}
	if err := c.client.Call(method, nil, out); err != nil {
		c.drop()
		return err
	}
	return nil
}

func (m *Model) refreshStatus() {
	if m.presence.dial() {
		var st daemon.PresenceStatus
		if m.presence.call(ipc.MethodStatus, &st) == nil {
			m.rpcStatus = st
		} else {
			m.rpcStatus = daemon.PresenceStatus{}
		}
	} else {
		m.rpcStatus = daemon.PresenceStatus{}
	}

	if m.botConn.dial() {
		var st daemon.BotStatus
		if m.botConn.call(ipc.MethodStatus, &st) == nil {
			m.botStatus = st
		} else {
			m.botStatus = daemon.BotStatus{}
		}
	} else {
		m.botStatus = daemon.BotStatus{}
	}
}

func (m *Model) ensureDaemons() tea.Cmd {
	return func() tea.Msg {
		if err := daemon.EnsureEnabled(); err != nil {
			m.log("daemon: %v", err)
		}
		return nil
	}
}
