package builder

import (
	"context"
	"testing"
	"time"

	"github.com/mxpv/podsync/pkg/feed"
	"github.com/mxpv/podsync/pkg/model"
	"github.com/mxpv/podsync/pkg/ytdl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSpotifyDownloader struct {
	playlist ytdl.Playlist
}

func (m mockSpotifyDownloader) PlaylistMetadata(_ context.Context, _ string) (ytdl.PlaylistMetadata, error) {
	return ytdl.PlaylistMetadata{}, nil
}

func (m mockSpotifyDownloader) Playlist(_ context.Context, _ string, _ int, _ model.Sorting) (ytdl.Playlist, error) {
	return m.playlist, nil
}

func TestSpotifyBuilder_Build(t *testing.T) {
	newer := time.Date(2024, 5, 10, 12, 0, 0, 0, time.UTC)
	older := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

	playlist := ytdl.Playlist{
		PlaylistMetadata: ytdl.PlaylistMetadata{
			Id:          "show123",
			Title:       "Test Show",
			Description: "Sample podcast",
			Thumbnails: []ytdl.PlaylistMetadataThumbnail{
				{Url: "http://thumb.small", Width: 64, Height: 64},
				{Url: "http://thumb.large", Width: 512, Height: 512},
			},
			Channel:    "Host Name",
			ChannelUrl: "https://open.spotify.com/show/show123",
			WebpageUrl: "https://open.spotify.com/show/show123",
		},
		Entries: []ytdl.PlaylistEntry{
			{
				ID:               "ep1",
				Title:            "Episode One",
				Description:      "First episode",
				Duration:         900,
				ReleaseTimestamp: older.Unix(),
				WebpageURL:       "https://open.spotify.com/episode/ep1",
				Thumbnail:        "http://thumb.ep1",
			},
			{
				ID:               "ep2",
				Title:            "Episode Two",
				Description:      "Second episode",
				Duration:         1200,
				ReleaseTimestamp: newer.Unix(),
				WebpageURL:       "https://open.spotify.com/episode/ep2",
				Thumbnails: []ytdl.PlaylistMetadataThumbnail{
					{Url: "http://thumb.ep2.small", Width: 64, Height: 64},
					{Url: "http://thumb.ep2.large", Width: 320, Height: 320},
				},
			},
		},
	}

	downloader := mockSpotifyDownloader{playlist: playlist}
	builder, err := NewSpotifyBuilder(downloader)
	require.NoError(t, err)

	cfg := &feed.Config{
		URL:          "https://open.spotify.com/show/show123",
		PageSize:     10,
		Quality:      model.QualityHigh,
		Format:       model.FormatVideo,
		PlaylistSort: model.SortingDesc,
	}

	result, err := builder.Build(context.Background(), cfg)
	require.NoError(t, err)

	assert.Equal(t, "Test Show", result.Title)
	assert.Equal(t, "Sample podcast", result.Description)
	assert.Equal(t, model.ProviderSpotify, result.Provider)
	assert.Equal(t, model.FormatAudio, result.Format)
	assert.Equal(t, "http://thumb.large", result.CoverArt)
	assert.Equal(t, "https://open.spotify.com/show/show123", result.ItemURL)

	if assert.Len(t, result.Episodes, 2) {
		assert.Equal(t, "Episode Two", result.Episodes[0].Title)
		assert.Equal(t, "ep2", result.Episodes[0].ID)
		assert.Equal(t, newer, result.Episodes[0].PubDate)
		assert.Equal(t, "http://thumb.ep2.large", result.Episodes[0].Thumbnail)
		assert.Greater(t, result.Episodes[0].Size, int64(0))
	}

	assert.Equal(t, result.PubDate, result.Episodes[0].PubDate)
}
