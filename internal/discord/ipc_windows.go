//go:build windows

package discord

import (
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

func dialIPC() (net.Conn, error) {
	var lastErr error
	timeout := 3 * time.Second
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf(`\\?\pipe\discord-ipc-%d`, i)
		conn, err := winio.DialPipe(path, &timeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no discord-ipc pipe found")
	}
	return nil, lastErr
}
