package ps4

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	GradePlatinum = 1
	GradeGold     = 2
	GradeSilver   = 3
	GradeBronze   = 4
)

type TrophyTitle struct {
	CommID       string
	Name         string
	Progress     int
	Unlocked     int
	Total        int
	Platinum     int
	Gold         int
	Silver       int
	Bronze       int
	LastUnlocked time.Time
}

type Trophy struct {
	ID           int
	GroupID      int
	Grade        int
	Unlocked     bool
	Hidden       bool
	TimeUnlocked time.Time
	Name         string
	Description  string
}

func (c *Client) trophyDB(get func(string) ([]byte, Meta, error)) ([]byte, Meta, error) {
	acc, err := c.AccountID()
	if err != nil {
		return nil, Meta{}, err
	}
	return get(fmt.Sprintf("/user/home/%s/trophy/db/trophy_local.db", acc))
}

func (c *Client) TrophyTitles() ([]TrophyTitle, Meta, error) {
	return c.trophyTitles(c.download)
}

func (c *Client) TrophyTitlesCached() ([]TrophyTitle, Meta, error) {
	return c.trophyTitles(c.downloadCached)
}

func (c *Client) trophyTitles(get func(string) ([]byte, Meta, error)) ([]TrophyTitle, Meta, error) {
	data, meta, err := c.trophyDB(get)
	if err != nil {
		return nil, meta, err
	}
	db, cleanup, err := openDB(data)
	if err != nil {
		return nil, meta, err
	}
	defer cleanup()
	titles, err := queryTrophyTitles(db)
	return titles, meta, err
}

func queryTrophyTitles(db *sql.DB) ([]TrophyTitle, error) {
	const q = `
SELECT trophy_title_id, title, progress, unlocked_trophy_num, trophy_num,
       platinum_num, gold_num, silver_num, bronze_num, time_last_unlocked
FROM tbl_trophy_title ORDER BY time_last_unlocked DESC`

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []TrophyTitle
	for rows.Next() {
		var t TrophyTitle
		var last string
		if err := rows.Scan(&t.CommID, &t.Name, &t.Progress, &t.Unlocked, &t.Total,
			&t.Platinum, &t.Gold, &t.Silver, &t.Bronze, &last); err != nil {
			return nil, err
		}
		t.LastUnlocked = parsePS4Time(last)
		titles = append(titles, t)
	}
	return titles, rows.Err()
}

func (c *Client) FindTitle(query string) (TrophyTitle, Meta, error) {
	titles, meta, err := c.TrophyTitles()
	if err != nil {
		return TrophyTitle{}, meta, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	for _, t := range titles {
		if strings.EqualFold(t.Name, query) || t.CommID == query {
			return t, meta, nil
		}
	}
	for _, t := range titles {
		if strings.Contains(strings.ToLower(t.Name), q) {
			return t, meta, nil
		}
	}
	return TrophyTitle{}, meta, errNoTitle
}

func (c *Client) Trophies(commID string) ([]Trophy, Meta, error) {
	data, meta, err := c.trophyDB(c.download)
	if err != nil {
		return nil, meta, err
	}
	db, cleanup, err := openDB(data)
	if err != nil {
		return nil, meta, err
	}
	defer cleanup()
	trophies, err := queryTrophies(db, commID)
	return trophies, meta, err
}

func queryTrophies(db *sql.DB, commID string) ([]Trophy, error) {
	const q = `
SELECT trophyid, groupid, grade, unlocked, hidden, time_unlocked, title, description
FROM tbl_trophy_flag WHERE trophy_title_id = ? ORDER BY groupid, trophyid`

	rows, err := db.Query(q, commID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trophies []Trophy
	for rows.Next() {
		var t Trophy
		var unlocked, hidden int
		var when string
		if err := rows.Scan(&t.ID, &t.GroupID, &t.Grade, &unlocked, &hidden, &when,
			&t.Name, &t.Description); err != nil {
			return nil, err
		}
		t.Unlocked = unlocked != 0
		t.Hidden = hidden != 0
		t.TimeUnlocked = parsePS4Time(when)
		trophies = append(trophies, t)
	}
	return trophies, rows.Err()
}

func (c *Client) SearchTitles(query string, limit int) ([]TrophyTitle, Meta, error) {
	titles, meta, err := c.TrophyTitlesCached()
	if err != nil {
		return nil, meta, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []TrophyTitle
	for _, t := range titles {
		if q == "" || strings.Contains(strings.ToLower(t.Name), q) {
			out = append(out, t)
			if len(out) >= limit {
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, meta, nil
}
