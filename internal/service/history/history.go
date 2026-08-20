package history

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
	lua "github.com/yuin/gopher-lua"
)

const (
	ftpPort     = "2121"
	remoteDir   = "/data/ps4rpc-go"
	remoteFile  = "history.lua"
	maxSessions = 500
)

func remotePath() string { return path.Join(remoteDir, remoteFile) }

type Session struct {
	TitleID  string
	GameName string
	Start    time.Time
	End      time.Time
}

func (s Session) Duration() time.Duration {
	end := s.End
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(s.Start)
}

func (s Session) Open() bool { return s.End.IsZero() }

const errNoIP historyError = "no console address configured"

type historyError string

func (e historyError) Error() string { return string(e) }

type Store struct {
	mu   sync.Mutex
	ip   string
	logf func(string, ...any)
}

func Open(logf func(string, ...any)) *Store {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Store{logf: logf}
}

func (st *Store) SetIP(ip string) {
	st.mu.Lock()
	st.ip = ip
	st.mu.Unlock()
}

func (st *Store) currentIP() string {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.ip
}

func (st *Store) EnsureSession(titleID, gameName string) Session {
	now := time.Now()
	ip := st.currentIP()
	if ip == "" {
		return Session{TitleID: titleID, GameName: gameName, Start: now}
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	sessions, err := fetch(ip)
	if err != nil {
		st.logf("history: %v", err)
		return Session{TitleID: titleID, GameName: gameName, Start: now}
	}

	if n := len(sessions); n > 0 {
		cur := &sessions[n-1]
		if cur.Open() {
			if cur.TitleID == titleID {
				if cur.GameName != gameName {
					cur.GameName = gameName
					if err := store(ip, sessions); err != nil {
						st.logf("history: %v", err)
					}
				}
				return *cur
			}
			cur.End = now
		}
	}

	sess := Session{TitleID: titleID, GameName: gameName, Start: now}
	sessions = append(sessions, sess)
	if len(sessions) > maxSessions {
		sessions = sessions[len(sessions)-maxSessions:]
	}
	if err := store(ip, sessions); err != nil {
		st.logf("history: %v", err)
	}
	return sess
}

func (st *Store) EndCurrent() {
	ip := st.currentIP()
	if ip == "" {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	sessions, err := fetch(ip)
	if err != nil {
		st.logf("history: %v", err)
		return
	}
	n := len(sessions)
	if n == 0 || !sessions[n-1].Open() {
		return
	}
	sessions[n-1].End = time.Now()
	if err := store(ip, sessions); err != nil {
		st.logf("history: %v", err)
	}
}

func Load(ip string) ([]Session, error) {
	if ip == "" {
		return nil, errNoIP
	}
	return fetch(ip)
}

func fetch(ip string) ([]Session, error) {
	data, err := download(ip, remotePath())
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	return decode(data)
}

func store(ip string, sessions []Session) error {
	return upload(ip, remotePath(), encode(sessions))
}

func decode(data []byte) ([]Session, error) {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(string(data)); err != nil {
		return nil, fmt.Errorf("history: %w", err)
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("history: history.lua must return a table")
	}

	var sessions []Session
	tbl.ForEach(func(_, v lua.LValue) {
		row, ok := v.(*lua.LTable)
		if !ok {
			return
		}
		s := Session{
			TitleID:  luaStr(row, "titleid"),
			GameName: luaStr(row, "name"),
		}
		if n, ok := luaInt64(row, "start"); ok {
			s.Start = time.Unix(n, 0)
		}
		if n, ok := luaInt64(row, "end"); ok {
			s.End = time.Unix(n, 0)
		}
		sessions = append(sessions, s)
	})
	return sessions, nil
}

func luaStr(t *lua.LTable, key string) string {
	v := t.RawGetString(key)
	if v == lua.LNil {
		return ""
	}
	return lua.LVAsString(v)
}

func luaInt64(t *lua.LTable, key string) (int64, bool) {
	n, ok := t.RawGetString(key).(lua.LNumber)
	if !ok {
		return 0, false
	}
	return int64(n), true
}

func encode(sessions []Session) []byte {
	var b strings.Builder
	b.WriteString("return {\n")
	for _, s := range sessions {
		fmt.Fprintf(&b, "    { titleid = %s, name = %s, start = %d",
			luaString(s.TitleID), luaString(s.GameName), s.Start.Unix())
		if !s.End.IsZero() {
			fmt.Fprintf(&b, ", [\"end\"] = %d", s.End.Unix())
		}
		b.WriteString(" },\n")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

func luaString(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n")
	return "\"" + r.Replace(s) + "\""
}

func dial(ip string) (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(net.JoinHostPort(ip, ftpPort),
		ftp.DialWithTimeout(6*time.Second), ftp.DialWithDisabledEPSV(true))
	if err != nil {
		return nil, err
	}
	if err := conn.Login("anonymous", "anonymous"); err != nil {
		conn.Quit()
		return nil, err
	}
	return conn, nil
}

func upload(ip, remote string, data []byte) error {
	conn, err := dial(ip)
	if err != nil {
		return err
	}
	defer conn.Quit()
	_ = conn.MakeDir(path.Dir(remote))
	return conn.Stor(remote, strings.NewReader(string(data)))
}

func download(ip, remote string) ([]byte, error) {
	conn, err := dial(ip)
	if err != nil {
		return nil, err
	}
	defer conn.Quit()

	resp, err := conn.Retr(remote)
	if err != nil {
		if isNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer resp.Close()
	return io.ReadAll(resp)
}

func isNotExist(err error) bool {
	var pe *textproto.Error
	if errors.As(err, &pe) {
		return pe.Code == ftp.StatusFileUnavailable
	}
	return strings.Contains(err.Error(), "550")
}
