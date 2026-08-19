package ps4

import (
	"encoding/json"
	"regexp"
	"strings"
)

const usersKey = "meta/users"

var hexDirRe = regexp.MustCompile(`^[0-9a-fA-F]{8}$`)

func (c *Client) AccountID() (string, error) {
	if c.account != "" {
		return c.account, nil
	}
	if users, ok := c.cachedUsers(); ok {
		return users[0], nil
	}
	users, err := c.Users()
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", errNoAccount
	}
	return users[0], nil
}

func (c *Client) Users() ([]string, error) {
	entries, err := c.nameList("/user/home")
	if err != nil {
		if users, ok := c.cachedUsers(); ok {
			return users, nil
		}
		return nil, err
	}
	var users []string
	for _, e := range entries {
		base := e[strings.LastIndex(e, "/")+1:]
		if hexDirRe.MatchString(base) {
			users = append(users, base)
		}
	}
	if raw, err := json.Marshal(users); err == nil {
		_ = c.store.Put(usersKey, raw)
	}
	return users, nil
}

func (c *Client) cachedUsers() ([]string, bool) {
	raw, _, ok := c.store.Get(usersKey)
	if !ok {
		return nil, false
	}
	var users []string
	if json.Unmarshal(raw, &users) != nil || len(users) == 0 {
		return nil, false
	}
	return users, true
}

func (c *Client) Username() (string, Meta, error) {
	acc, err := c.AccountID()
	if err != nil {
		return "", Meta{}, err
	}
	data, meta, err := c.download("/user/home/" + acc + "/username.dat")
	if err != nil {
		return "", meta, err
	}
	if i := indexZero(data); i >= 0 {
		data = data[:i]
	}
	return strings.TrimSpace(string(data)), meta, nil
}

func indexZero(b []byte) int {
	for i, ch := range b {
		if ch == 0 {
			return i
		}
	}
	return -1
}

type ps4Error string

func (e ps4Error) Error() string { return string(e) }

const (
	errNoAccount ps4Error = "no user account folder found under /user/home"
	errNoTitle   ps4Error = "no matching game with trophies"
)
