package ps4

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jlaffaye/ftp"
)

func (c *Client) Icon(titleID string) ([]byte, error) {
	return c.download(fmt.Sprintf("/user/appmeta/%s/icon0.png", titleID))
}

func (c *Client) Avatar() ([]byte, error) {
	acc, err := c.AccountID()
	if err != nil {
		return nil, err
	}
	base := "/system_data/priv/cache/profile/0x" + strings.ToUpper(acc) + "/"
	for _, name := range []string{"avatar.png", "picture.png"} {
		if data, err := c.download(base + name); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	return nil, errNoAvatar
}

func (c *Client) LatestScreenshot() (name string, data []byte, err error) {

	roots := []string{"/user/av_contents/photo"}
	var best *ftp.Entry
	var bestDir string
	for _, root := range roots {
		best, bestDir = c.walkForCapture(root, best, bestDir, 0)
	}
	if best == nil {
		return "", nil, errNoCapture
	}
	data, err = c.retr(bestDir + "/" + best.Name)
	if err != nil {
		return "", nil, err
	}
	return best.Name, data, nil
}

const (
	maxCaptureBytes = 24 * 1024 * 1024
	minCaptureBytes = 50 * 1024
)

func isCapture(e *ftp.Entry) bool {
	if e.Size < minCaptureBytes || e.Size > maxCaptureBytes {
		return false
	}
	n := strings.ToLower(e.Name)
	return strings.HasSuffix(n, ".jpg") || strings.HasSuffix(n, ".jpeg") || strings.HasSuffix(n, ".png")
}

func (c *Client) walkForCapture(dir string, best *ftp.Entry, bestDir string, depth int) (*ftp.Entry, string) {
	if depth > 5 {
		return best, bestDir
	}
	entries, err := c.list(dir)
	if err != nil {
		return best, bestDir
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Time.After(entries[j].Time) })
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		switch e.Type {
		case ftp.EntryTypeFile:
			if isCapture(e) && (best == nil || e.Time.After(best.Time)) {
				best, bestDir = e, dir
			}
		case ftp.EntryTypeFolder:
			best, bestDir = c.walkForCapture(dir+"/"+e.Name, best, bestDir, depth+1)
		}
	}
	return best, bestDir
}

const errNoCapture ps4Error = "no screenshots or clips found"
const errNoAvatar ps4Error = "no avatar image found"
