package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
)

type Server struct {
	name string
	ln   net.Listener
	h    Handler

	mu     sync.Mutex
	conns  map[*serverConn]bool
	closed bool
}

type serverConn struct {
	c  net.Conn
	mu sync.Mutex
}

func (sc *serverConn) send(f frame) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	sc.mu.Lock()
	defer sc.mu.Unlock()
	_, err = sc.c.Write(b)
	return err
}

func Serve(name string, h Handler) (*Server, error) {
	ln, err := listen(name)
	if err != nil {
		return nil, err
	}
	s := &Server{name: name, ln: ln, h: h, conns: map[*serverConn]bool{}}
	go s.accept()
	return s, nil
}

func (s *Server) accept() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		sc := &serverConn{c: c}
		s.add(sc)
		go s.handle(sc)
	}
}

func (s *Server) handle(sc *serverConn) {
	defer s.drop(sc)

	sr := bufio.NewScanner(sc.c)
	sr.Buffer(make([]byte, 0, 4096), 1<<20)
	for sr.Scan() {
		var req request
		if err := json.Unmarshal(sr.Bytes(), &req); err != nil {
			log.Printf("ipc: %s: malformed frame: %v", s.name, err)
			continue
		}
		if req.Method == MethodSubscribe {
			s.subscribe(sc)
		}
		res, err := s.dispatch(req)
		out := frame{ID: req.ID}
		if err != nil {
			out.Error = err.Error()
		} else if out.Result, err = marshal(res); err != nil {
			out.Error = err.Error()
		}
		if sc.send(out) != nil {
			return
		}
	}
}

func (s *Server) dispatch(req request) (any, error) {
	switch req.Method {
	case MethodPing:
		return "pong", nil
	case MethodSubscribe:
		return nil, nil
	}
	if s.h == nil {
		return nil, errors.New("ipc: no handler")
	}
	return s.h(req.Method, req.Params)
}

func (s *Server) add(sc *serverConn) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		_ = sc.c.Close()
		return
	}
	s.conns[sc] = false
	s.mu.Unlock()
}

func (s *Server) subscribe(sc *serverConn) {
	s.mu.Lock()
	if _, ok := s.conns[sc]; ok {
		s.conns[sc] = true
	}
	s.mu.Unlock()
}

func (s *Server) drop(sc *serverConn) {
	s.mu.Lock()
	delete(s.conns, sc)
	s.mu.Unlock()
	_ = sc.c.Close()
}

func (s *Server) Broadcast(event string, payload any) {
	b, err := marshal(payload)
	if err != nil {
		return
	}
	s.mu.Lock()
	subs := make([]*serverConn, 0, len(s.conns))
	for sc, subscribed := range s.conns {
		if subscribed {
			subs = append(subs, sc)
		}
	}
	s.mu.Unlock()

	f := frame{Event: event, Payload: b}
	for _, sc := range subs {
		if sc.send(f) != nil {
			s.drop(sc)
		}
	}
}

func (s *Server) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	open := make([]*serverConn, 0, len(s.conns))
	for sc := range s.conns {
		open = append(open, sc)
	}
	s.conns = map[*serverConn]bool{}
	s.mu.Unlock()

	err := s.ln.Close()
	for _, sc := range open {
		_ = sc.c.Close()
	}
	cleanup(s.name)
	return err
}
