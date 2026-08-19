package discord

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const handshakeTimeout = 5 * time.Second

const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
)

type Activity struct {
	Name       string
	Details    string
	LargeImage string
	LargeText  string
	Start      int64
}

type Client struct {
	conn net.Conn
}

func Connect(clientID string) (*Client, error) {
	conn, err := dialIPC()
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn}
	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	payload, _ := json.Marshal(map[string]any{"v": 1, "client_id": clientID})
	if err := c.send(opHandshake, payload); err != nil {
		conn.Close()
		return nil, err
	}
	if _, _, err := c.recv(); err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return c, nil
}

func (c *Client) Update(a Activity) error {
	act := map[string]any{}
	if a.Name != "" {
		act["name"] = a.Name
	}
	if a.Details != "" {
		act["details"] = a.Details
	}
	assets := map[string]any{}
	if a.LargeImage != "" {
		assets["large_image"] = a.LargeImage
	}
	if a.LargeText != "" {
		assets["large_text"] = a.LargeText
	}
	if len(assets) > 0 {
		act["assets"] = assets
	}
	if a.Start != 0 {
		act["timestamps"] = map[string]any{"start": a.Start}
	}
	return c.setActivity(act)
}

func (c *Client) Clear() error {
	return c.setActivity(nil)
}

func (c *Client) setActivity(activity any) error {
	payload, _ := json.Marshal(map[string]any{
		"cmd": "SET_ACTIVITY",
		"args": map[string]any{
			"pid":      os.Getpid(),
			"activity": activity,
		},
		"nonce": strconv.FormatInt(time.Now().UnixNano(), 10),
	})
	if err := c.send(opFrame, payload); err != nil {
		return err
	}
	_, _, err := c.recv()
	return err
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	_ = c.send(opClose, []byte("{}"))
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) send(op int32, payload []byte) error {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], uint32(op))
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))
	if _, err := c.conn.Write(append(header, payload...)); err != nil {
		return err
	}
	return nil
}

func (c *Client) recv() (int32, []byte, error) {
	header := make([]byte, 8)
	if _, err := readFull(c.conn, header); err != nil {
		return 0, nil, err
	}
	op := int32(binary.LittleEndian.Uint32(header[0:4]))
	length := binary.LittleEndian.Uint32(header[4:8])
	payload := make([]byte, length)
	if _, err := readFull(c.conn, payload); err != nil {
		return op, nil, err
	}
	return op, payload, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func ConnectRetry(clientID string) *Client {
	c, _ := ConnectRetryCtx(context.Background(), clientID, nil)
	return c
}

func ConnectRetryCtx(ctx context.Context, clientID string, logf func(string, ...any)) (*Client, error) {
	if logf == nil {
		logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c, err := Connect(clientID)
		if err == nil {
			logf("Connected to Discord client")
			return c, nil
		}
		logf("! Could not find Discord running. '%v'.", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Second):
		}
	}
}
