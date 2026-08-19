package ps4

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"time"
)

func PlaytimePath(titleID string) string {
	return "/data/GoldHEN/stats/" + titleID + ".ini"
}

func (c *Client) Playtime(titleID string) (played time.Duration, starts int, ok bool) {
	data, _, err := c.download(PlaytimePath(titleID))
	if err != nil {
		return 0, 0, false
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		k, v, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "Time_Played":
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				played = time.Duration(n) * time.Second
				ok = true
			}
		case "Game_Starts":
			starts, _ = strconv.Atoi(v)
		}
	}
	return played, starts, ok
}
