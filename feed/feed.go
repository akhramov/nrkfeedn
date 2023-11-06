package feed

import (
	"time"

	"github.com/eduncan911/podcast"
)

type Podcast struct {
	p *podcast.Podcast
}

type Episode struct {
	Title       string
	Description string
	Link        string
	Image       string
	Date        time.Time
}

func New(title, subtitle, image string) Podcast {
	p := podcast.New(title, "", subtitle, nil, nil)
	p.AddImage(image)

	return Podcast{
		p: &p,
	}
}

func (p *Podcast) AddEpisode(episode Episode) error {
	item := podcast.Item{
		Title:       episode.Title,
		Link:        episode.Link,
		Description: episode.Description,
		PubDate:     &episode.Date,
	}

	item.AddImage(episode.Image)
	_, err := p.p.AddItem(item)

	return err
}

func (p *Podcast) GetFeed() string {
	return p.p.String()
}
