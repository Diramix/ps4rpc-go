package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/ipc"
)

const watchInterval = 3 * time.Second

type service interface {
	status() any
	reload(cfg *config.Config)
	stop()
}

type syncedService struct {
	mu  sync.Mutex
	svc service
}

func (s *syncedService) status() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.svc.status()
}

func (s *syncedService) reload(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.svc.reload(cfg)
}

func (s *syncedService) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.svc.stop()
}

func serve(role string, logs *logbuf, svc service) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	stop := make(chan struct{})
	var stopOnce sync.Once
	requestStop := func() { stopOnce.Do(func() { close(stop) }) }

	svc = &syncedService{svc: svc}

	srv, err := ipc.Serve(role, func(method string, _ json.RawMessage) (any, error) {
		switch method {
		case ipc.MethodStatus:
			return svc.status(), nil
		case ipc.MethodLogs:
			return logs.history(), nil
		case ipc.MethodReload:
			fresh, _, err := loadConfig()
			if err != nil {
				return nil, err
			}
			svc.reload(fresh)
			st := svc.status()
			if !Wanted(fresh, role) {
				logs.printf("%s: disabled in the config", role)
				requestStop()
			}
			return st, nil
		case ipc.MethodShutdown:
			requestStop()
			return "ok", nil
		}
		return nil, fmt.Errorf("daemon: unknown method %q", method)
	})
	if err != nil {
		return err
	}
	logs.attach(srv)
	logs.printf("%s: daemon started (pid %d)", role, os.Getpid())

	svc.reload(cfg)
	if !Wanted(cfg, role) {
		logs.printf("%s: disabled in the config, nothing to do", role)
		requestStop()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go watchConfig(ctx, role, stop, requestStop, logs, svc)

	select {
	case <-ctx.Done():
	case <-stop:
	}

	logs.printf("%s: shutting down", role)
	svc.stop()
	return srv.Close()
}

func watchConfig(ctx context.Context, role string, stop <-chan struct{}, requestStop func(), logs *logbuf, svc service) {
	known := config.Fingerprint()
	var last *config.Config
	t := time.NewTicker(watchInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-t.C:
		}
		fp := config.Fingerprint()
		if fp == known {
			continue
		}
		cfg, existed, err := config.Load()
		known = config.Fingerprint()
		if err != nil || !existed {
			continue
		}
		if cfg.Equal(last) {
			continue
		}
		last = cfg
		logs.printf("config: reloaded from disk")
		svc.reload(cfg)
		if !Wanted(cfg, role) {
			logs.printf("%s: disabled in the config", role)
			requestStop()
			return
		}
	}
}

func loadConfig() (*config.Config, bool, error) {
	config.SetDir(config.DefaultDir())
	return config.Load()
}
