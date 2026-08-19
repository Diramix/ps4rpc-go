package ps4

import (
	"os"
	"testing"
)

func TestLive(t *testing.T) {
	ip := os.Getenv("PS4_LIVE_IP")
	if ip == "" {
		t.Skip("set PS4_LIVE_IP to run")
	}
	c := New(ip)

	acc, err := c.AccountID()
	t.Logf("account=%q err=%v", acc, err)

	st, err := c.Status()
	t.Logf("status=%+v err=%v", st, err)

	games, _, err := c.Games()
	if err != nil {
		t.Fatalf("games: %v", err)
	}
	t.Logf("games=%d total=%d bytes", len(games), TotalSize(games))

	titles, _, err := c.TrophyTitles()
	if err != nil {
		t.Fatalf("titles: %v", err)
	}
	t.Logf("trophy titles=%d, first=%q", len(titles), func() string {
		if len(titles) > 0 {
			return titles[0].Name
		}
		return ""
	}())

	if tt, _, err := c.FindTitle("Last of Us"); err == nil {
		tr, _, _ := c.Trophies(tt.CommID)
		t.Logf("FindTitle -> %q %d/%d, trophies loaded=%d", tt.Name, tt.Unlocked, tt.Total, len(tr))
	} else {
		t.Logf("FindTitle err=%v", err)
	}

	if up, err := c.Uptime(); err == nil {
		t.Logf("uptime=%s", up)
	} else {
		t.Logf("uptime err=%v", err)
	}
}
