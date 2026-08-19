package ps4

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"

	"ps4rpc/internal/app/cache"
)

const ftpPort = "2121"

type Client struct {
	ip          string
	account     string
	dialTimeout time.Duration
	cacheTTL    time.Duration

	store *cache.Store

	mu           sync.Mutex
	cache        map[string]cachedFile
	offlineUntil time.Time
}

const offlineBackoff = 30 * time.Second

type cachedFile struct {
	data    []byte
	fetched time.Time
	meta    Meta
}

type Meta struct {
	Fetched time.Time
	Live    bool
}

func liveMeta() Meta { return Meta{Fetched: time.Now(), Live: true} }

func (m Meta) Stale() bool { return !m.Live && !m.Fetched.IsZero() }

func (m Meta) Merge(o Meta) Meta {
	switch {
	case m.Fetched.IsZero():
		return o
	case o.Fetched.IsZero():
		return m
	case o.Stale() && !m.Stale():
		return o
	case m.Stale() && !o.Stale():
		return m
	case o.Fetched.Before(m.Fetched):
		return o
	}
	return m
}

func New(ip string) *Client {
	return &Client{
		ip:          ip,
		dialTimeout: 6 * time.Second,
		cacheTTL:    2 * time.Minute,
		cache:       map[string]cachedFile{},
	}
}

func (c *Client) IP() string { return c.ip }

func (c *Client) SetAccount(id string) { c.account = id }

func (c *Client) SetCache(store *cache.Store) { c.store = store }

func (c *Client) Cache() *cache.Store { return c.store }

func (c *Client) unreachable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(c.offlineUntil)
}

func (c *Client) markReachable(reachable bool) {
	c.mu.Lock()
	if reachable {
		c.offlineUntil = time.Time{}
	} else {
		c.offlineUntil = time.Now().Add(offlineBackoff)
	}
	c.mu.Unlock()
}

func (c *Client) dial() (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(net.JoinHostPort(c.ip, ftpPort),
		ftp.DialWithTimeout(c.dialTimeout),
		ftp.DialWithDisabledEPSV(true))
	if err != nil {
		c.markReachable(false)
		return nil, err
	}
	c.markReachable(true)
	if err := conn.Login("anonymous", "anonymous"); err != nil {
		conn.Quit()
		return nil, err
	}
	return conn, nil
}

func (c *Client) nameList(dir string) ([]string, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()
	return conn.NameList(dir)
}

func (c *Client) list(dir string) ([]*ftp.Entry, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()
	return conn.List(dir)
}

func (c *Client) retr(remote string) ([]byte, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Quit()
	resp, err := conn.Retr(remote)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) download(remote string) ([]byte, Meta, error) {
	return c.fetch(remote, false)
}

func (c *Client) downloadCached(remote string) ([]byte, Meta, error) {
	return c.fetch(remote, true)
}

func (c *Client) fetch(remote string, cacheFirst bool) ([]byte, Meta, error) {
	c.mu.Lock()
	if cf, ok := c.cache[remote]; ok && time.Since(cf.fetched) < c.cacheTTL {
		c.mu.Unlock()
		return cf.data, cf.meta, nil
	}
	c.mu.Unlock()

	if cacheFirst || c.unreachable() {
		if data, fetched, ok := c.store.Get(remote); ok {
			return data, Meta{Fetched: fetched}, nil
		}
	}

	data, err := c.retr(remote)
	if err != nil {
		if data, fetched, ok := c.store.Get(remote); ok {
			return data, Meta{Fetched: fetched}, nil
		}
		return nil, Meta{}, err
	}
	_ = c.store.Put(remote, data)

	meta := liveMeta()
	c.mu.Lock()
	c.cache[remote] = cachedFile{data: data, fetched: time.Now(), meta: meta}
	c.mu.Unlock()
	return data, meta, nil
}
