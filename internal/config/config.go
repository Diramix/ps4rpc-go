package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

var (
	Dir        = "config"
	Path       = filepath.Join(Dir, "main.lua")
	RpcPath    = filepath.Join(Dir, "rpc.lua")
	BotPath    = filepath.Join(Dir, "bot.lua")
	DevPath    = filepath.Join(Dir, "dev.lua")
	MappedPath = filepath.Join(Dir, "mapped.lua")
)

func DataDir() string {
	const app = "ps4rpc-go"
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, app)
		}
	} else {
		if base := os.Getenv("XDG_DATA_HOME"); base != "" {
			return filepath.Join(base, app)
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", app)
		}
	}
	return "."
}

func DefaultDir() string {
	return filepath.Join(DataDir(), "config")
}

func SetDir(dir string) {
	Dir = dir
	Path = filepath.Join(Dir, "main.lua")
	RpcPath = filepath.Join(Dir, "rpc.lua")
	BotPath = filepath.Join(Dir, "bot.lua")
	DevPath = filepath.Join(Dir, "dev.lua")
	MappedPath = filepath.Join(Dir, "mapped.lua")
}

type Core struct {
	IP       string
	ClientID int64
	WaitTime int
	Enabled  bool
}

type DevApp struct {
	DevID   string
	TitleID string
}

type Bot struct {
	Token     string
	OwnerID   string
	GuildID   string
	AccountID string
	Enabled   bool
}

type Mapped struct {
	TitleID string `json:"titleid"`
	Name    string `json:"name"`
	Image   string `json:"image"`
}

type Config struct {
	Core    Core
	Bot     Bot
	Devapps []DevApp
	Mapped  []Mapped
}

func Load() (*Config, bool, error) {
	if _, err := os.Stat(Path); err != nil {
		if os.IsNotExist(err) {
			return Default(), false, nil
		}
		return Default(), false, err
	}

	L := lua.NewState()
	defer L.Close()
	setLuaPath(L)

	if err := L.DoFile(Path); err != nil {
		return Default(), true, err
	}
	ret := L.Get(-1)
	root, ok := ret.(*lua.LTable)
	if !ok {
		return Default(), true, fmt.Errorf("config: %s must return a table", Path)
	}

	cfg := Default()
	fallbackDevapps := cfg.Devapps
	cfg.Devapps = nil
	cfg.applyCore(coreTable(root))
	cfg.applyRpc(tableField(root, "rpc"))
	cfg.applyBot(tableField(root, "bot"))
	cfg.applyDev(tableField(root, "dev"))
	cfg.applyMapped(tableField(root, "mapped"))

	if cfg.Devapps == nil {
		cfg.Devapps = fallbackDevapps
	}

	if err := cfg.migrate(); err != nil {
		return cfg, true, err
	}
	return cfg, true, nil
}

func setLuaPath(L *lua.LState) {
	pkg := L.GetGlobal("package").(*lua.LTable)
	pattern := filepath.Join(Dir, "?.lua")
	cur := lua.LVAsString(pkg.RawGetString("path"))
	pkg.RawSetString("path", lua.LString(pattern+";"+cur))
}

func tableField(t *lua.LTable, name string) *lua.LTable {
	if t == nil {
		return nil
	}
	if v, ok := t.RawGetString(name).(*lua.LTable); ok {
		return v
	}
	return nil
}

func coreTable(root *lua.LTable) *lua.LTable {
	if t := tableField(root, "core"); t != nil {
		return t
	}
	return tableField(root, "var")
}

func (c *Config) applyCore(t *lua.LTable) {
	if t == nil {
		return
	}
	v := &c.Core
	v.IP = luaStr(t, "ip", v.IP)
}

func (c *Config) applyRpc(t *lua.LTable) {
	if t == nil {
		return
	}
	v := &c.Core
	v.ClientID = luaInt64(t, "client_id", v.ClientID)
	v.WaitTime = luaInt(t, "wait_time", v.WaitTime)
	v.Enabled = luaBool(t, "enabled", v.Enabled)
}

func (c *Config) applyBot(t *lua.LTable) {
	if t == nil {
		return
	}
	b := &c.Bot
	b.Token = luaStr(t, "token", b.Token)
	b.OwnerID = luaStr(t, "owner_id", b.OwnerID)
	b.GuildID = luaStr(t, "guild_id", b.GuildID)
	b.AccountID = luaStr(t, "account_id", b.AccountID)
	b.Enabled = luaBool(t, "enabled", b.Enabled)
}

func (c *Config) applyDev(t *lua.LTable) {
	if t == nil {
		return
	}
	t.ForEach(func(_, v lua.LValue) {
		row, ok := v.(*lua.LTable)
		if !ok {
			return
		}
		c.Devapps = append(c.Devapps, DevApp{
			DevID:   luaStr(row, "devid", ""),
			TitleID: luaStr(row, "titleid", ""),
		})
	})
}

func (c *Config) applyMapped(t *lua.LTable) {
	if t == nil {
		return
	}
	t.ForEach(func(_, v lua.LValue) {
		row, ok := v.(*lua.LTable)
		if !ok {
			return
		}
		c.Mapped = append(c.Mapped, Mapped{
			TitleID: luaStr(row, "titleid", ""),
			Name:    luaStr(row, "name", ""),
			Image:   luaStr(row, "image", ""),
		})
	})
}

func (c *Config) migrate() error {
	for path, want := range c.files() {
		raw, err := os.ReadFile(path)
		if err != nil || string(raw) != want {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				return err
			}
		}
	}
	_ = os.Remove(filepath.Join(Dir, "var.lua"))
	return nil
}

func (c *Config) files() map[string]string {
	return map[string]string{
		Path:       c.renderMain(),
		RpcPath:    c.renderRpc(),
		BotPath:    c.renderBot(),
		DevPath:    c.renderDev(),
		MappedPath: c.renderMapped(),
	}
}

func (c *Config) renderMain() string {
	var b strings.Builder
	b.WriteString("return {\n")
	b.WriteString("    core = {\n")
	fmt.Fprintf(&b, "        ip = %s,\n", luaString(c.Core.IP))
	b.WriteString("    },\n")
	b.WriteString("    rpc = require(\"rpc\"),\n")
	b.WriteString("    bot = require(\"bot\"),\n")
	b.WriteString("    dev = require(\"dev\"),\n")
	b.WriteString("    mapped = require(\"mapped\"),\n")
	b.WriteString("}\n")
	return b.String()
}

func (c *Config) renderRpc() string {
	var b strings.Builder
	b.WriteString("return {\n")
	fmt.Fprintf(&b, "    client_id = %q,\n", strconv.FormatInt(c.Core.ClientID, 10))
	fmt.Fprintf(&b, "    wait_time = %d,\n", c.Core.WaitTime)
	fmt.Fprintf(&b, "    enabled = %t,\n", c.Core.Enabled)
	b.WriteString("}\n")
	return b.String()
}

func (c *Config) renderBot() string {
	var b strings.Builder
	b.WriteString("return {\n")
	fmt.Fprintf(&b, "    token = %s,\n", luaString(c.Bot.Token))
	fmt.Fprintf(&b, "    owner_id = %s,\n", luaString(c.Bot.OwnerID))
	fmt.Fprintf(&b, "    guild_id = %s,\n", luaString(c.Bot.GuildID))
	fmt.Fprintf(&b, "    account_id = %s,\n", luaString(c.Bot.AccountID))
	fmt.Fprintf(&b, "    enabled = %t,\n", c.Bot.Enabled)
	b.WriteString("}\n")
	return b.String()
}

func (c *Config) renderDev() string {
	var b strings.Builder
	b.WriteString("return {\n")
	for _, d := range c.Devapps {
		fmt.Fprintf(&b, "    { devid = %s, titleid = %s },\n", luaString(d.DevID), luaString(d.TitleID))
	}
	b.WriteString("}\n")
	return b.String()
}

func (c *Config) renderMapped() string {
	var b strings.Builder
	b.WriteString("return {\n")
	for _, m := range c.Mapped {
		fmt.Fprintf(&b, "    { titleid = %s, name = %s, image = %s },\n",
			luaString(m.TitleID), luaString(m.Name), luaString(m.Image))
	}
	b.WriteString("}\n")
	return b.String()
}

func (c *Config) Save() error {
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return err
	}
	for path, want := range c.files() {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) AppendMapped(m Mapped) error {
	c.Mapped = append(c.Mapped, m)
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(MappedPath, []byte(c.renderMapped()), 0o644)
}

func luaString(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n")
	return "\"" + r.Replace(s) + "\""
}

func luaStr(t *lua.LTable, key, def string) string {
	v := t.RawGetString(key)
	if v == lua.LNil {
		return def
	}
	return lua.LVAsString(v)
}

func luaBool(t *lua.LTable, key string, def bool) bool {
	switch val := t.RawGetString(key).(type) {
	case lua.LBool:
		return bool(val)
	case lua.LString:
		if b, err := strconv.ParseBool(strings.TrimSpace(string(val))); err == nil {
			return b
		}
	}
	return def
}

func luaInt(t *lua.LTable, key string, def int) int {
	return int(luaInt64(t, key, int64(def)))
}

func luaInt64(t *lua.LTable, key string, def int64) int64 {
	v := t.RawGetString(key)
	switch val := v.(type) {
	case lua.LNumber:
		return int64(val)
	case lua.LString:
		if n, err := strconv.ParseInt(strings.TrimSpace(string(val)), 10, 64); err == nil {
			return n
		}
	}
	return def
}

func (c *Config) Clone() *Config {
	out := *c
	out.Devapps = append([]DevApp(nil), c.Devapps...)
	out.Mapped = append([]Mapped(nil), c.Mapped...)
	return &out
}

func (c *Config) Equal(o *Config) bool {
	if o == nil || c.Core != o.Core || c.Bot != o.Bot ||
		len(c.Devapps) != len(o.Devapps) || len(c.Mapped) != len(o.Mapped) {
		return false
	}
	for i := range c.Devapps {
		if c.Devapps[i] != o.Devapps[i] {
			return false
		}
	}
	for i := range c.Mapped {
		if c.Mapped[i] != o.Mapped[i] {
			return false
		}
	}
	return true
}

func Fingerprint() string {
	var b strings.Builder
	for _, p := range []string{Path, RpcPath, BotPath, DevPath, MappedPath} {
		fi, err := os.Stat(p)
		if err != nil {
			b.WriteString(p + ":-;")
			continue
		}
		fmt.Fprintf(&b, "%s:%d:%d;", p, fi.Size(), fi.ModTime().UnixNano())
	}
	return b.String()
}
