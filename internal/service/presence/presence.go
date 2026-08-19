package presence

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/source/discord"
	"ps4rpc/internal/source/gameinfo"
	"ps4rpc/internal/source/ps4"
)

type Status struct {
	Running   bool
	PS4Online bool
	DiscordOK bool
	TitleID   string
	GameName  string
}

type Service struct {
	cfg  *config.Config
	logf func(string, ...any)

	mu     sync.Mutex
	status Status
	cancel context.CancelFunc
	done   chan struct{}
	rpc    *discord.Client
}

func New(cfg *config.Config, logf func(string, ...any)) *Service {
	if logf == nil {
		logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	return &Service{cfg: cfg.Clone(), logf: logf}
}

func (s *Service) SetConfig(cfg *config.Config) {
	s.mu.Lock()
	s.cfg = cfg.Clone()
	s.mu.Unlock()
}

func (s *Service) conf() *config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) setStatus(f func(*Status)) {
	s.mu.Lock()
	f(&s.status)
	s.mu.Unlock()
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("rpc: already running")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.status.Running = true
	done := s.done
	s.mu.Unlock()

	go func() {
		defer close(done)
		s.run(ctx)
		s.setStatus(func(st *Status) {
			st.Running = false
			st.DiscordOK = false
		})
	}()
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}
}

func (s *Service) Run(ctx context.Context) {
	s.setStatus(func(st *Status) { st.Running = true })
	defer s.setStatus(func(st *Status) { st.Running = false })
	s.run(ctx)
}

func (s *Service) run(ctx context.Context) {
	rpc, err := discord.ConnectRetryCtx(ctx, clientID(s.conf().Core.ClientID), s.logf)
	if err != nil {
		return
	}
	s.setRPC(rpc)
	defer s.closeRPC()
	s.setStatus(func(st *Status) { st.DiscordOK = true })

	prevTitleID := ""
	devAppChanged := false
	for {
		if ctx.Err() != nil {
			return
		}

		titleID, gameType, ok := ps4.New(s.conf().Core.IP).TitleID()
		if !ok {
			s.setStatus(func(st *Status) { st.PS4Online = false })
			s.logf("ps4: console not found, sleeping")
			for {
				err := ps4.New(s.conf().Core.IP).Check()
				if err == nil {
					break
				}
				s.logf("ps4: %v", err)
				if !s.sleep(ctx) {
					return
				}
			}
			continue
		}
		s.setStatus(func(st *Status) {
			st.PS4Online = true
			st.TitleID = titleID
		})
		s.logf("ps4: title %q (%s)", titleID, gameType)

		if prevTitleID == titleID {
			s.logf("reusing previous presence data")
		} else {
			name, image := s.checkMapped(titleID, gameType)
			prevTitleID = titleID
			s.setStatus(func(st *Status) { st.GameName = name })

			devAppChanged = s.changeDevApp(ctx, titleID, devAppChanged)

			if titleID == "main_menu" {
				_ = s.currentRPC().Clear()
			} else {
				s.closeRPC()
				c, err := discord.ConnectRetryCtx(ctx, clientID(s.conf().Core.ClientID), s.logf)
				if err != nil {
					return
				}
				s.setRPC(c)
				s.updatePresence(ctx, name, image, titleID)
			}
		}
		if !s.sleep(ctx) {
			return
		}
	}
}

func (s *Service) sleep(ctx context.Context) bool {
	wait := s.conf().Core.WaitTime
	if wait <= 0 {
		wait = 30
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Duration(wait) * time.Second):
		return true
	}
}

func (s *Service) currentRPC() *discord.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rpc
}

func (s *Service) setRPC(c *discord.Client) {
	s.mu.Lock()
	s.rpc = c
	s.status.DiscordOK = c != nil
	s.mu.Unlock()
}

func (s *Service) closeRPC() {
	c := s.currentRPC()
	if c != nil {
		_ = c.Close()
	}
	s.setRPC(nil)
}

func (s *Service) checkMapped(titleID, gameType string) (name, image string) {
	image = titleID
	for _, m := range s.conf().Mapped {
		if m.TitleID == titleID {
			s.logf("check_mapped():  (%q, %q)", m.Name, m.Image)
			return m.Name, m.Image
		}
	}
	if titleID == "main_menu" {
		return "", titleID
	}

	s.logf("check_mapped():  game has not been mapped yet.")
	switch gameType {
	case "PS4":
		name, image = gameinfo.GetPS4GameInfo(titleID)
	case "PS1/2":
		name, image = gameinfo.GetClassicGameInfo(titleID)
	default:
		name, image = gameinfo.GetOtherGameInfo(titleID)
	}
	if err := s.conf().AppendMapped(config.Mapped{TitleID: titleID, Name: name, Image: image}); err != nil {
		s.logf("check_mapped():  %v", err)
	}
	return name, image
}

func (s *Service) changeDevApp(ctx context.Context, titleID string, changed bool) bool {
	for _, app := range s.conf().Devapps {
		if app.TitleID == titleID && app.TitleID != "" {
			s.logf("change_dev_app():    changing to new developer app")
			s.closeRPC()
			c, err := discord.ConnectRetryCtx(ctx, app.DevID, s.logf)
			if err != nil {
				return changed
			}
			s.setRPC(c)
			return true
		}
	}
	if changed {
		s.logf("change_dev_app():    reverting to default developer app")
		s.closeRPC()
		c, err := discord.ConnectRetryCtx(ctx, clientID(s.conf().Core.ClientID), s.logf)
		if err != nil {
			return false
		}
		s.setRPC(c)
		return false
	}
	return changed
}

func (s *Service) updatePresence(ctx context.Context, name, image, titleID string) {
	a := discord.Activity{LargeImage: image, LargeText: titleID, Name: name}
	c := s.currentRPC()
	if c == nil {
		return
	}
	if err := c.Update(a); err != nil {
		s.logf("Error with Discord: %v", err)
		s.closeRPC()
		if n, err := discord.ConnectRetryCtx(ctx, clientID(s.conf().Core.ClientID), s.logf); err == nil {
			s.setRPC(n)
		}
	}
}

func clientID(id int64) string {
	return strconv.FormatInt(id, 10)
}
