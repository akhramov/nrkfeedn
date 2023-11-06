package database

import (
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/types"
)

type Database struct {
	db *genji.DB
}

type Podcast struct {
	Id       string
	Title    string
	Subtitle string
	Image    string
}

type Episode struct {
	Title       string
	Description string
	Link        string
	Image       string
	Date        int64
	PodcastId   string `genji:"podcast_id"`
}

func Open(url string) (*Database, error) {
	db, err := genji.Open(url)
	if err != nil {
		return nil, err
	}

	err = db.Exec("CREATE TABLE IF NOT EXISTS podcasts (id text PRIMARY KEY, title text, subtitle text, image text)")
	if err != nil {
		db.Close()
		return nil, err
	}

	err = db.Exec("CREATE TABLE IF NOT EXISTS episodes (title text, description text, link text, image text, date integer, podcast_id text)")
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

func (db *Database) GetPodcast(id string) (*Podcast, error) {
	d, err := db.db.QueryDocument("SELECT * FROM podcasts WHERE id = ?", id)
	if err != nil {
		return nil, err
	}

	var p Podcast
	if err := document.StructScan(d, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

func (db *Database) GetPodcasts() ([]Podcast, error) {
	result := make([]Podcast, 0)
	stream, err := db.db.Query("SELECT * FROM podcasts")
	if err != nil {
		return result, err
	}
	defer stream.Close()

	err = stream.Iterate(func(d types.Document) error {
		var p Podcast

		if err := document.StructScan(d, &p); err != nil {
			return err
		}

		result = append(result, p)
		return nil
	})

	return result, err
}

func (db *Database) GetEpisodes(id string) ([]Episode, error) {
	result := make([]Episode, 0)
	stream, err := db.db.Query("SELECT * FROM episodes WHERE podcast_id = ?", id)
	if err != nil {
		return result, err
	}
	defer stream.Close()

	err = stream.Iterate(func(d types.Document) error {
		var ep Episode

		if err := document.StructScan(d, &ep); err != nil {
			return err
		}

		result = append(result, ep)
		return nil
	})

	return result, err
}

func (db *Database) LastEpisodeTime(id string) (int64, error) {
	d, err := db.db.QueryDocument("SELECT * FROM episodes WHERE podcast_id = ? ORDER BY date DESC LIMIT 1", id)
	if genji.IsNotFoundError(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	var ep Episode
	if err := document.StructScan(d, &ep); err != nil {
		return 0, err
	}

	return ep.Date, nil
}

func (db *Database) CreatePodcast(p *Podcast) error {
	return db.db.Exec("INSERT INTO podcasts VALUES ?", p)
}

func (db *Database) CreateEpisode(ep *Episode) error {
	return db.db.Exec("INSERT INTO episodes VALUES ?", ep)
}

func (db *Database) Close() {
	db.db.Close()
}
