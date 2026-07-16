package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	Dir        = "config"
	Path       = filepath.Join(Dir, "ps4rpc.conf")
	MappedPath = filepath.Join(Dir, "mapped.json")
)

// SetDir points the config package at a different directory, recomputing the
// file paths derived from it.
func SetDir(dir string) {
	Dir = dir
	Path = filepath.Join(Dir, "ps4rpc.conf")
	MappedPath = filepath.Join(Dir, "mapped.json")
}

type Var struct {
	IP            string
	ClientID      int64
	WaitTime      int
	RetroCovers   bool
	Hibernate     bool
	HibernateTime int
	UseDevapps    bool
	ShowTimer     bool
	UseAppname    bool
}

type DevApp struct {
	DevID   string
	TitleID string
}

type Mapped struct {
	TitleID string `json:"titleid"`
	Name    string `json:"name"`
	Image   string `json:"image"`
}

type Config struct {
	Var     Var
	Devapps []DevApp
	Mapped  []Mapped
}

func Default() *Config {
	return &Config{
		Var: Var{
			ClientID:      858345055966461973,
			WaitTime:      30,
			RetroCovers:   true,
			HibernateTime: 600,
		},
		Devapps: []DevApp{{}},
		Mapped:  []Mapped{},
	}
}

func Load() (*Config, bool, error) {
	f, err := os.Open(Path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), false, nil
		}
		return Default(), false, err
	}
	defer f.Close()

	cfg := Default()
	cfg.Devapps = nil

	var block string
	var kv map[string]string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, "{") {
			block = strings.TrimSpace(strings.TrimSuffix(line, "{"))
			kv = map[string]string{}
			continue
		}
		if line == "}" {
			cfg.apply(block, kv)
			block, kv = "", nil
			continue
		}
		if kv != nil {
			if k, v, ok := strings.Cut(line, "="); ok {
				kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, true, err
	}
	if cfg.Devapps == nil {
		cfg.Devapps = []DevApp{{}}
	}
	if err := cfg.loadMapped(); err != nil {
		return cfg, true, err
	}
	return cfg, true, nil
}

func (c *Config) loadMapped() error {
	data, err := os.ReadFile(MappedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &c.Mapped)
}

func (c *Config) apply(block string, kv map[string]string) {
	switch block {
	case "var":
		v := &c.Var
		v.IP = kv["ip"]
		v.ClientID = atoi64(kv["client_id"], v.ClientID)
		v.WaitTime = atoi(kv["wait_time"], v.WaitTime)
		v.RetroCovers = atob(kv["retro_covers"], v.RetroCovers)
		v.Hibernate = atob(kv["hibernate"], v.Hibernate)
		v.HibernateTime = atoi(kv["hibernate_time"], v.HibernateTime)
		v.UseDevapps = atob(kv["use_devapps"], v.UseDevapps)
		v.ShowTimer = atob(kv["show_timer"], v.ShowTimer)
		v.UseAppname = atob(kv["use_appname"], v.UseAppname)
	case "devapp":
		c.Devapps = append(c.Devapps, DevApp{DevID: kv["devid"], TitleID: kv["titleid"]})
	}
}

func (c *Config) Save() error {
	var b strings.Builder
	b.WriteString("var {\n")
	fmt.Fprintf(&b, "    ip = %s\n", c.Var.IP)
	fmt.Fprintf(&b, "    client_id = %d\n", c.Var.ClientID)
	fmt.Fprintf(&b, "    wait_time = %d\n", c.Var.WaitTime)
	fmt.Fprintf(&b, "    retro_covers = %t\n", c.Var.RetroCovers)
	fmt.Fprintf(&b, "    hibernate = %t\n", c.Var.Hibernate)
	fmt.Fprintf(&b, "    hibernate_time = %d\n", c.Var.HibernateTime)
	fmt.Fprintf(&b, "    use_devapps = %t\n", c.Var.UseDevapps)
	fmt.Fprintf(&b, "    show_timer = %t\n", c.Var.ShowTimer)
	fmt.Fprintf(&b, "    use_appname = %t\n", c.Var.UseAppname)
	b.WriteString("}\n")

	for _, d := range c.Devapps {
		b.WriteString("\ndevapp {\n")
		fmt.Fprintf(&b, "    devid = %s\n", d.DevID)
		fmt.Fprintf(&b, "    titleid = %s\n", d.TitleID)
		b.WriteString("}\n")
	}

	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(Path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return c.saveMapped()
}

func (c *Config) saveMapped() error {
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return err
	}
	mapped := c.Mapped
	if mapped == nil {
		mapped = []Mapped{}
	}
	data, err := json.MarshalIndent(mapped, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(MappedPath, append(data, '\n'), 0o644)
}

func (c *Config) AppendMapped(m Mapped) error {
	c.Mapped = append(c.Mapped, m)
	return c.saveMapped()
}

func atoi(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

func atoi64(s string, def int64) int64 {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	return def
}

func atob(s string, def bool) bool {
	switch strings.ToLower(s) {
	case "true", "yes", "1":
		return true
	case "false", "no", "0":
		return false
	}
	return def
}
