package ps4

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jlaffaye/ftp"
)

func IconPath(titleID string) string {
	return fmt.Sprintf("/user/appmeta/%s/icon0.png", titleID)
}

func (c *Client) Icon(titleID string) ([]byte, Meta, error) {
	return c.download(IconPath(titleID))
}

func (c *Client) Avatar() ([]byte, Meta, error) {
	acc, err := c.AccountID()
	if err != nil {
		return nil, Meta{}, err
	}
	base := "/system_data/priv/cache/profile/0x" + strings.ToUpper(acc) + "/"
	for _, name := range []string{"avatar.png", "picture.png"} {
		if data, meta, err := c.download(base + name); err == nil && len(data) > 0 {
			return data, meta, nil
		}
	}
	return nil, Meta{}, errNoAvatar
}

func (c *Client) LatestScreenshot() (name string, data []byte, meta Meta, err error) {
	roots := []string{"/user/av_contents/photo"}
	var best *ftp.Entry
	var bestDir string
	for _, root := range roots {
		best, bestDir = c.walkForCapture(root, best, bestDir, 0)
	}
	if best == nil {
		if name, data, meta, ok := c.cachedScreenshot(); ok {
			return name, data, meta, nil
		}
		return "", nil, Meta{}, errNoCapture
	}
	data, err = c.retr(bestDir + "/" + best.Name)
	if err != nil {
		if name, data, meta, ok := c.cachedScreenshot(); ok {
			return name, data, meta, nil
		}
		return "", nil, Meta{}, err
	}
	_ = c.store.Put(screenshotKey, data)
	_ = c.store.Put(screenshotNameKey, []byte(best.Name))
	return best.Name, data, liveMeta(), nil
}

const (
	screenshotKey     = "meta/last_screenshot"
	screenshotNameKey = "meta/last_screenshot.name"
)

func (c *Client) cachedScreenshot() (name string, data []byte, meta Meta, ok bool) {
	data, fetched, ok := c.store.Get(screenshotKey)
	if !ok {
		return "", nil, Meta{}, false
	}
	name = "screenshot.jpg"
	if raw, _, ok := c.store.Get(screenshotNameKey); ok && len(raw) > 0 {
		name = string(raw)
	}
	return name, data, Meta{Fetched: fetched}, true
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
