package gameinfo

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

//go:embed data/ps1_games.md data/ps2_games.md
var gameLists embed.FS

var tmdbKey = mustHex("F5DE66D2680E255B2DF79E74F890EBF349262F618BCAE2A9ACCDEE5156CE8DF2CDF2D48C71173CDC2594465B87405D197CF1AED3B7E9671EEB56CA6753C2E6B0")

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func get(url string) ([]byte, bool) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}
	return body, true
}

type tmdbResponse struct {
	Names []struct {
		Name string `json:"name"`
	} `json:"names"`
	Icons []struct {
		Icon string `json:"icon"`
	} `json:"icons"`
}

func GetPS4GameInfo(titleID string) (name, image string) {
	tid := titleID + "_00"
	mac := hmac.New(sha1.New, tmdbKey)
	mac.Write([]byte(tid))
	sum := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	url := fmt.Sprintf("https://tmdb.np.dl.playstation.net/tmdb2/%s_%s/%s.json", tid, sum, tid)

	body, ok := get(url)
	if ok {
		var r tmdbResponse
		if err := json.Unmarshal(body, &r); err == nil && len(r.Names) > 0 && len(r.Icons) > 0 {
			return r.Names[0].Name, r.Icons[0].Icon
		}
	}
	log.Printf("GetPS4GameInfo():     No entry found in TMDB for %s", titleID)
	return titleID, strings.ToLower(titleID)
}

func GetClassicGameInfo(titleID string) (name, image string) {
	image = strings.ToLower(titleID)
	if n, ok := searchClassic("data/ps1_games.md", titleID); ok {
		return n, image
	}
	if n, ok := searchClassic("data/ps2_games.md", titleID); ok {
		return n, image
	}
	return "Unknown", image
}

func searchClassic(path, titleID string) (string, bool) {
	f, err := gameLists.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, titleID) {
			continue
		}
		parts := strings.SplitN(line, ";", 2)
		if len(parts) < 2 {
			continue
		}
		return strings.TrimRight(parts[1], " \r"), true
	}
	return "", false
}

func GetOtherGameInfo(titleID string) (name, image string) {
	return titleID, strings.ToLower(titleID)
}
