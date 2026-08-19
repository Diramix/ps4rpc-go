package ps4

import (
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"time"
)

const appDBPath = "/system_data/priv/mms/app.db"

type Game struct {
	TitleID    string
	Name       string
	Category   string
	Size       int64
	LastPlayed time.Time
}

func (g Game) IsGame() bool {
	return strings.HasPrefix(g.Category, "gd")
}

func (c *Client) Library() ([]Game, Meta, error) {
	return c.library(c.download)
}

func (c *Client) LibraryCached() ([]Game, Meta, error) {
	return c.library(c.downloadCached)
}

func (c *Client) library(get func(string) ([]byte, Meta, error)) ([]Game, Meta, error) {
	data, meta, err := get(appDBPath)
	if err != nil {
		return nil, meta, err
	}
	db, cleanup, err := openDB(data)
	if err != nil {
		return nil, meta, err
	}
	defer cleanup()
	games, err := queryLibrary(db)
	return games, meta, err
}

func queryLibrary(db *sql.DB) ([]Game, error) {
	const q = `
SELECT titleId,
  MAX(CASE WHEN key='TITLE'              THEN val END) AS name,
  MAX(CASE WHEN key='CATEGORY'           THEN val END) AS cat,
  MAX(CASE WHEN key='#_size'             THEN val END) AS size,
  MAX(CASE WHEN key='#_last_access_time' THEN val END) AS last
FROM tbl_appinfo GROUP BY titleId`

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var titleID string
		var name, cat, size, last *string
		if err := rows.Scan(&titleID, &name, &cat, &size, &last); err != nil {
			return nil, err
		}
		g := Game{TitleID: titleID}
		if name != nil {
			g.Name = *name
		}
		if cat != nil {
			g.Category = *cat
		}
		if size != nil {
			g.Size, _ = strconv.ParseInt(*size, 10, 64)
		}
		if last != nil {
			g.LastPlayed = parsePS4Time(*last)
		}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sortByName(games)
	return games, nil
}

func sortByName(games []Game) {
	keys := make([]string, len(games))
	for i, g := range games {
		keys[i] = strings.ToLower(g.Name)
	}
	sort.Sort(&byNameKey{games: games, keys: keys})
}

type byNameKey struct {
	games []Game
	keys  []string
}

func (b *byNameKey) Len() int      { return len(b.games) }
func (b *byNameKey) Swap(i, j int) { b.games[i], b.games[j] = b.games[j], b.games[i]; b.keys[i], b.keys[j] = b.keys[j], b.keys[i] }
func (b *byNameKey) Less(i, j int) bool { return b.keys[i] < b.keys[j] }

func (c *Client) Games() ([]Game, Meta, error) {
	return games(c.Library())
}

func (c *Client) GamesCached() ([]Game, Meta, error) {
	return games(c.LibraryCached())
}

func games(all []Game, meta Meta, err error) ([]Game, Meta, error) {
	if err != nil {
		return nil, meta, err
	}
	games := all[:0]
	for _, g := range all {
		if g.IsGame() && g.Name != "" {
			games = append(games, g)
		}
	}
	return games, meta, nil
}

func TotalSize(games []Game) int64 {
	var total int64
	for _, g := range games {
		total += g.Size
	}
	return total
}

func parsePS4Time(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02 15:04:05.000", "2006-01-02 15:04:05", "2006-01-02T15:04:05.00Z", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
