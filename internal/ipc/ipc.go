package ipc

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	MethodStatus    = "status"
	MethodReload    = "reload"
	MethodShutdown  = "shutdown"
	MethodSubscribe = "subscribe"
	MethodLogs      = "logs"
	MethodPing      = "ping"

	EventLog = "log"
)

const dialTimeout = 3 * time.Second

type Handler func(method string, params json.RawMessage) (any, error)

type request struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type frame struct {
	ID      uint64          `json:"id,omitempty"`
	Event   string          `json:"event,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type Event struct {
	Name    string
	Payload json.RawMessage
}

func (e Event) Str() string {
	var s string
	if err := json.Unmarshal(e.Payload, &s); err != nil {
		return string(e.Payload)
	}
	return s
}

func Ping(name string) bool {
	c, err := Dial(name)
	if err != nil {
		return false
	}
	defer c.Close()
	return c.Call(MethodPing, nil, nil) == nil
}

func marshal(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("ipc: marshal: %w", err)
	}
	return b, nil
}
