package ps4

import (
	"bufio"
	"net"
	"regexp"
	"strconv"
	"time"
)

const klogPort = "3232"

var elapsedRe = regexp.MustCompile(`system timer elapsed (\d+):(\d+):(\d+)`)

func (c *Client) Uptime() (time.Duration, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(c.ip, klogPort), c.dialTimeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(4 * time.Second))

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 8192), 64*1024)
	var last time.Duration
	found := false
	for scanner.Scan() {
		if m := elapsedRe.FindStringSubmatch(scanner.Text()); m != nil {
			h, _ := strconv.Atoi(m[1])
			mi, _ := strconv.Atoi(m[2])
			s, _ := strconv.Atoi(m[3])
			last = time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute + time.Duration(s)*time.Second
			found = true
			break
		}
	}
	if !found {
		return 0, errNoUptime
	}
	return last, nil
}

const errNoUptime ps4Error = "no uptime line seen in klog stream"
