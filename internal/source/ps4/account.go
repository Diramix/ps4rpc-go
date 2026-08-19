package ps4

import (
	"regexp"
	"strings"
)

var hexDirRe = regexp.MustCompile(`^[0-9a-fA-F]{8}$`)

func (c *Client) AccountID() (string, error) {
	if c.account != "" {
		return c.account, nil
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
		return nil, err
	}
	var users []string
	for _, e := range entries {
		base := e[strings.LastIndex(e, "/")+1:]
		if hexDirRe.MatchString(base) {
			users = append(users, base)
		}
	}
	return users, nil
}

func (c *Client) Username() (string, error) {
	acc, err := c.AccountID()
	if err != nil {
		return "", err
	}
	data, err := c.download("/user/home/" + acc + "/username.dat")
	if err != nil {
		return "", err
	}
	if i := indexZero(data); i >= 0 {
		data = data[:i]
	}
	return strings.TrimSpace(string(data)), nil
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
