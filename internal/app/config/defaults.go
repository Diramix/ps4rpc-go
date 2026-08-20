package config

import (
	"embed"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

//go:embed defaults/*.lua
var defaults embed.FS

var modules = []string{"activity", "bot", "cache", "dev", "mapped"}

func Default() *Config {
	cfg, err := loadDefaults()
	if err != nil {
		panic(err)
	}
	return cfg
}

func loadDefaults() (*Config, error) {
	L := lua.NewState()
	defer L.Close()

	for _, name := range modules {
		src, err := defaults.ReadFile("defaults/" + name + ".lua")
		if err != nil {
			return nil, fmt.Errorf("config: defaults: %w", err)
		}
		L.PreloadModule(name, func(L *lua.LState) int {
			fn, err := L.LoadString(string(src))
			if err != nil {
				L.RaiseError("%v", err)
			}
			L.Push(fn)
			L.Call(0, 1)
			return 1
		})
	}

	src, err := defaults.ReadFile("defaults/main.lua")
	if err != nil {
		return nil, fmt.Errorf("config: defaults: %w", err)
	}
	if err := L.DoString(string(src)); err != nil {
		return nil, fmt.Errorf("config: defaults: %w", err)
	}
	root, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("config: defaults/main.lua must return a table")
	}

	cfg := &Config{}
	cfg.applyCore(coreTable(root))
	cfg.applyActivity(activityTable(root))
	cfg.applyBot(tableField(root, "bot"))
	cfg.applyCache(tableField(root, "cache"))
	cfg.applyDev(tableField(root, "dev"))
	cfg.applyMapped(tableField(root, "mapped"))
	if cfg.Mapped == nil {
		cfg.Mapped = []Mapped{}
	}
	return cfg, nil
}
