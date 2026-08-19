package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"sync"
)

var ErrClosed = errors.New("ipc: connection closed")

type Client struct {
	conn net.Conn

	wmu sync.Mutex

	mu      sync.Mutex
	nextID  uint64
	pending map[uint64]chan frame
	closed  bool

	events chan Event
}

func Dial(name string) (*Client, error) {
	conn, err := dial(name)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:    conn,
		pending: map[uint64]chan frame{},
		events:  make(chan Event, 256),
	}
	go c.read()
	return c, nil
}

func (c *Client) read() {
	defer c.shutdown()
	sr := bufio.NewScanner(c.conn)
	sr.Buffer(make([]byte, 0, 4096), 1<<20)
	for sr.Scan() {
		var f frame
		if err := json.Unmarshal(sr.Bytes(), &f); err != nil {
			continue
		}
		if f.Event != "" {
			select {
			case c.events <- Event{Name: f.Event, Payload: f.Payload}:
			default:
			}
			continue
		}
		c.mu.Lock()
		ch, ok := c.pending[f.ID]
		delete(c.pending, f.ID)
		c.mu.Unlock()
		if ok {
			ch <- f
		}
	}
}

func (c *Client) shutdown() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	pending := c.pending
	c.pending = map[uint64]chan frame{}
	c.mu.Unlock()

	for id, ch := range pending {
		ch <- frame{ID: id, Error: ErrClosed.Error()}
	}
	_ = c.conn.Close()
	close(c.events)
}

func (c *Client) Call(method string, params, out any) error {
	raw, err := marshal(params)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	c.nextID++
	id := c.nextID
	ch := make(chan frame, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	b, err := json.Marshal(request{ID: id, Method: method, Params: raw})
	if err != nil {
		return err
	}
	c.wmu.Lock()
	_, err = c.conn.Write(append(b, '\n'))
	c.wmu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return err
	}

	f := <-ch
	if f.Error != "" {
		return errors.New(f.Error)
	}
	if out == nil || len(f.Result) == 0 {
		return nil
	}
	return json.Unmarshal(f.Result, out)
}

func (c *Client) Events() <-chan Event { return c.events }

func (c *Client) Close() error {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return nil
	}
	return c.conn.Close()
}
