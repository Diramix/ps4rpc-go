package daemon

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"ps4rpc/internal/service/ipc"
)

const maxLogLines = 500

type logbuf struct {
	mu    sync.Mutex
	lines []string
	srv   *ipc.Server
}

func (l *logbuf) attach(srv *ipc.Server) {
	l.mu.Lock()
	l.srv = srv
	l.mu.Unlock()
}

func (l *logbuf) printf(format string, args ...any) {
	l.add(fmt.Sprintf(format, args...))
}

func (l *logbuf) add(s string) {
	l.mu.Lock()
	srv := l.srv
	var fresh []string
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		l.lines = append(l.lines, line)
		fresh = append(fresh, line)
	}
	if len(l.lines) > maxLogLines {
		l.lines = l.lines[len(l.lines)-maxLogLines:]
	}
	l.mu.Unlock()

	fmt.Println(strings.Join(fresh, "\n"))
	if srv != nil {
		for _, line := range fresh {
			srv.Broadcast(ipc.EventLog, line)
		}
	}
}

func (l *logbuf) history() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.lines...)
}

func (l *logbuf) Write(p []byte) (int, error) {
	l.add(string(p))
	return len(p), nil
}

func (l *logbuf) captureStdLog() {
	log.SetFlags(0)
	log.SetOutput(io.Writer(l))
}
