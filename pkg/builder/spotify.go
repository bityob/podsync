package builder

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"time"

	"github.com/mxpv/podsync/pkg/feed"
	"github.com/mxpv/podsync/pkg/model"
	"github.com/mxpv/podsync/pkg/ytdl"
	"github.com/pkg/errors"
)

const (
	spotifyHighAudioBytesPerSecond = 128000 / 8
	spotifyLowAudioBytesPerSecond  = 48000 / 8
)

type SpotifyBuilder struct {
	downloader Downloader
}

func NewSpotifyBuilder(downloader Downloader) (*SpotifyBuilder, error) {
	if downloader == nil {
		return nil, errors.New("downloader is required")
	}

	return &SpotifyBuilder{downloader: downloader}, nil
}

func (s *SpotifyBuilder) Build(ctx context.Context, cfg *feed.Config) (*model.Feed, error) {
	info, err := ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	pageSize := cfg.PageSize
	if pageSize == 0 {
		pageSize = model.DefaultPageSize
	}

	format := cfg.Format
	if format == model.FormatVideo {
		// Spotify podcasts are audio-only, ensure we pick an audio workflow
		format = model.FormatAudio
	}

	feed := &model.Feed{
		ItemID:          info.ItemID,
		Provider:        info.Provider,
		LinkType:        info.LinkType,
		Format:          format,
		Quality:         cfg.Quality,
		CoverArtQuality: cfg.Custom.CoverArtQuality,
		PageSize:        pageSize,
		PlaylistSort:    cfg.PlaylistSort,
		PrivateFeed:     cfg.PrivateFeed,
		UpdatedAt:       time.Now().UTC(),
	}

	playlist, err := s.downloader.Playlist(ctx, cfg.URL, feed.PageSize, feed.PlaylistSort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spotify playlist")
	}

	feed.Title = playlist.Title
	feed.Description = playlist.Description
	feed.Author = playlist.Channel
	if feed.Author == "" {
		feed.Author = playlist.ChannelId
	}
	feed.ItemURL = playlist.WebpageUrl
	if feed.ItemURL == "" {
		feed.ItemURL = cfg.URL
	}
	if feed.ItemID == "" {
		feed.ItemID = playlist.Id
	}
	if feed.Description == "" {
		feed.Description = feed.Title
	}

	feed.CoverArt = selectSpotifyThumbnail(playlist.Thumbnails)

	for idx, entry := range playlist.Entries {
		videoURL := entry.WebpageURL
		if videoURL == "" {
			videoURL = entry.URL
		}
		if videoURL == "" {
			continue
		}

		duration := int64(entry.Duration)
		if duration == 0 && entry.Duration > 0 {
			duration = int64(entry.Duration)
		}

		pubDate := parseSpotifyDate(entry)
		thumb := entry.Thumbnail
		if thumb == "" {
			thumb = selectSpotifyThumbnail(entry.Thumbnails)
		}

		feed.Episodes = append(feed.Episodes, &model.Episode{
			ID:          spotifyEpisodeID(entry, idx),
			Title:       entry.Title,
			Description: entry.Description,
			Thumbnail:   thumb,
			Duration:    duration,
			Size:        estimateSpotifySize(duration, feed),
			VideoURL:    videoURL,
			PubDate:     pubDate,
			Status:      model.EpisodeNew,
		})
	}

	sort.Slice(feed.Episodes, func(i, j int) bool {
		if feed.PlaylistSort == model.SortingDesc {
			return feed.Episodes[i].PubDate.After(feed.Episodes[j].PubDate)
		}
		return feed.Episodes[i].PubDate.Before(feed.Episodes[j].PubDate)
	})

	if len(feed.Episodes) > feed.PageSize {
		feed.Episodes = feed.Episodes[:feed.PageSize]
	}

	for i := range feed.Episodes {
		feed.Episodes[i].Order = fmt.Sprintf("%d", i)
	}

	if feed.CoverArt == "" && len(feed.Episodes) > 0 {
		feed.CoverArt = feed.Episodes[0].Thumbnail
	}

	if len(feed.Episodes) > 0 {
		feed.PubDate = feed.Episodes[0].PubDate
		for _, episode := range feed.Episodes[1:] {
			if episode.PubDate.After(feed.PubDate) {
				feed.PubDate = episode.PubDate
			}
		}
	}

	return feed, nil
}

func selectSpotifyThumbnail(thumbnails []ytdl.PlaylistMetadataThumbnail) string {
	if len(thumbnails) == 0 {
		return ""
	}

	copyThumbs := make([]ytdl.PlaylistMetadataThumbnail, len(thumbnails))
	copy(copyThumbs, thumbnails)

	sort.Slice(copyThumbs, func(i, j int) bool {
		return copyThumbs[i].Width < copyThumbs[j].Width
	})

	return copyThumbs[len(copyThumbs)-1].Url
}

func estimateSpotifySize(duration int64, feed *model.Feed) int64 {
	if duration == 0 {
		return 0
	}

	if feed.Quality == model.QualityLow {
		return duration * spotifyLowAudioBytesPerSecond
	}

	return duration * spotifyHighAudioBytesPerSecond
}

func parseSpotifyDate(entry ytdl.PlaylistEntry) time.Time {
	switch {
	case entry.ReleaseTimestamp != 0:
		return time.Unix(entry.ReleaseTimestamp, 0).UTC()
	case entry.Timestamp != 0:
		return time.Unix(entry.Timestamp, 0).UTC()
	}

	if t := parseSpotifyDateString(entry.ReleaseDate); !t.IsZero() {
		return t
	}

	if t := parseSpotifyDateString(entry.UploadDate); !t.IsZero() {
		return t
	}

	return time.Now().UTC()
}

func parseSpotifyDateString(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	layouts := []string{"20060102", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}

	return time.Time{}
}

func spotifyEpisodeID(entry ytdl.PlaylistEntry, fallback int) string {
	if entry.ID != "" {
		return entry.ID
	}

	if entry.WebpageURL != "" {
		return fmt.Sprintf("%x", md5.Sum([]byte(entry.WebpageURL)))
	}

	if entry.URL != "" {
		return fmt.Sprintf("%x", md5.Sum([]byte(entry.URL)))
	}

	return fmt.Sprintf("episode-%d", fallback)
}
