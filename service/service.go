package service

import (
	"context"
	"errors"
	"time"

	"github.com/akhramov/fattigmanns_nrk_radio/database"
	"github.com/akhramov/fattigmanns_nrk_radio/feed"
	"github.com/akhramov/fattigmanns_nrk_radio/playback"
	"github.com/akhramov/fattigmanns_nrk_radio/psapi"

	log "github.com/sirupsen/logrus"
)

const (
	psapiURL = "https://psapi.nrk.no/"
)

type Service struct {
	db             *database.Database
	psapiClient    *psapi.ClientWithResponses
	playbackClient *playback.ClientWithResponses
	done           chan bool
}

func New() (*Service, error) {
	psapiClient, err := psapi.NewClientWithResponses(psapiURL)
	if err != nil {
		return nil, err
	}

	playbackClient, err := playback.NewClientWithResponses(psapiURL)
	if err != nil {
		return nil, err
	}

	// TODO: make this configurable
	db, err := database.Open("Database")
	if err != nil {
		return nil, err
	}

	service := Service{
		psapiClient:    psapiClient,
		playbackClient: playbackClient,
		db:             db,
		done:           make(chan bool),
	}

	done := make(chan bool)
	ticker := time.NewTicker(time.Minute * 5)

	go func() {
		defer ticker.Stop()

		for {
			err = service.updatePodcasts()

			if err != nil {
				log.Warn("failed to update podcasts: ", err)
			} else {
				log.Info("updated podcasts!")
			}

			select {
			case <-done:
				service.db.Close()
				return
			case <-ticker.C:
				continue
			}
		}
	}()

	return &service, nil
}

func (s *Service) CreatePodcast(id string) error {
	params := psapi.GetPodcastParams{}
	resp, err := s.psapiClient.GetPodcastWithResponse(context.Background(), id, &params)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return errors.New(resp.HTTPResponse.Status)
	}
	series := resp.JSON200.Series
	subtitle := ""
	if series.Titles.Subtitle != nil {
		subtitle = *series.Titles.Subtitle
	}
	image := ""
	if len(series.PosterImage) > 0 {
		image = series.PosterImage[0].Url
	}

	podcast := database.Podcast{
		Id:       id,
		Title:    series.Titles.Title,
		Subtitle: subtitle,
		Image:    image,
	}

	return s.db.CreatePodcast(&podcast)
}

func (s *Service) GetFeed(id string) (string, error) {
	p, err := s.db.GetPodcast(id)
	if err != nil {
		return "", err
	}

	podcast := feed.New(p.Title, p.Subtitle, p.Image)

	episodes, err := s.db.GetEpisodes(id)
	if err != nil {
		return "", err
	}

	for _, ep := range episodes {
		err := podcast.AddEpisode(feed.Episode{
			Title:       ep.Title,
			Description: ep.Description,
			Link:        ep.Link,
			Image:       ep.Image,
			Date:        time.Unix(ep.Date, 0),
		})

		if err != nil {
			return "", err
		}
	}

	return podcast.GetFeed(), err
}

func (s *Service) updatePodcasts() error {
	podcasts, err := s.db.GetPodcasts()
	if err != nil {
		return err
	}

	for _, podcast := range podcasts {
		err := s.updateEpisodes(podcast.Id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) updateEpisodes(id string) error {
	lastEpisodeTime, err := s.db.LastEpisodeTime(id)
	if err != nil {
		return err
	}
	limit := time.Unix(lastEpisodeTime, 0)
	log.Info("The limit for ", id, " is ", limit)
	finished := false
	page := 1
	pageSize := 10
	for {
		params := psapi.GetPodcastepisodesParams{
			PageSize: &pageSize,
			Page:     &page,
		}

		resp, err := s.psapiClient.GetPodcastepisodesWithResponse(context.Background(), id, &params)
		if err != nil {
			return err
		}

		if resp.HTTPResponse.StatusCode != 200 {
			return errors.New(resp.HTTPResponse.Status)
		}

		episodes := *resp.JSON200.Embedded.Episodes

		for _, ep := range episodes {
			url, err := episodeMedia(ep.EpisodeId)
			if err != nil {
				return err
			}

			images := make([]string, len(*ep.Image))

			for idx, val := range *ep.Image {
				images[idx] = val.Url
			}
			description := ""
			if ep.Titles.Subtitle != nil {
				description = *ep.Titles.Subtitle
			}

			time, err := time.Parse(time.RFC3339, ep.Date)
			if err != nil {
				return err
			}

			if time.Before(limit) || time.Equal(limit) {
				finished = true
				break
			}

			err = s.db.CreateEpisode(&database.Episode{
				Title:       ep.Titles.Title,
				Description: description,
				Link:        url,
				Image:       images[0],
				Date:        time.Unix(),
				PodcastId:   id,
			})

			if err != nil {
				return err
			}
		}

		page++

		if len(episodes) < 10 || finished {
			break
		}
	}

	return nil
}

func episodeMedia(id string) (string, error) {
	c, err := playback.NewClientWithResponses("https://psapi.nrk.no/")
	if err != nil {
		return "", err
	}
	params := playback.GetPlaybackManifestRedirectParams{}
	resp, err := c.GetPlaybackManifestRedirectWithResponse(context.Background(), id, &params)
	if err != nil {
		return "", err
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return "", errors.New(resp.HTTPResponse.Status)
	}
	response := *resp.JSON200
	metadata, err := response.AsPlayableManifest()
	if err != nil {
		return "", err
	}

	return metadata.Playable.Assets[0].Url, nil
}
