package ps4info

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
)

const ftpPort = "2121"

type Client struct {
	ip          string
	account     string
	dialTimeout time.Duration
	cacheTTL    time.Duration

	mu    sync.Mutex
	cache map[string]cachedFile
}

type cachedFile struct {
	data    []byte
	fetched time.Time
}

func New(ip string) *Client {
	return &Client{
		ip:          ip,
		dialTimeout: 6 * time.Second,
		cacheTTL:    20 * time.Second,
		cache:       map[string]cachedFile{},
	}
}

func (c *Client) IP() string { return c.ip }

func (c *Client) SetAccount(id string) { c.account = id }

func (c *Client) dial() (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(net.JoinHostPort(c.ip, ftpPort),
		ftp.DialWithTimeout(c.dialTimeout),
		ftp.DialWithDisabledEPSV(true))
	if err != nil {
		return nil, err
	}
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

func (c *Client) download(remote string) ([]byte, error) {
	c.mu.Lock()
	if cf, ok := c.cache[remote]; ok && time.Since(cf.fetched) < c.cacheTTL {
		c.mu.Unlock()
		return cf.data, nil
	}
	c.mu.Unlock()

	data, err := c.retr(remote)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[remote] = cachedFile{data: data, fetched: time.Now()}
	c.mu.Unlock()
	return data, nil
}
