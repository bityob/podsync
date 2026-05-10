package ytdl

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mxpv/podsync/pkg/feed"
	"github.com/mxpv/podsync/pkg/model"

	"github.com/stretchr/testify/assert"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name         string
		format       model.Format
		customFormat feed.CustomFormat
		quality      model.Quality
		maxHeight    int
		output       string
		videoURL     string
		ytdlArgs     []string
		expect       []string
	}{
		{
			name:     "Audio unknown quality",
			format:   model.FormatAudio,
			output:   "/tmp/1",
			videoURL: "http://url",
			expect:   []string{"--extract-audio", "--audio-format", "mp3", "--format", "bestaudio", "--output", "/tmp/1", "http://url"},
		},
		{
			name:     "Audio low quality",
			format:   model.FormatAudio,
			quality:  model.QualityLow,
			output:   "/tmp/1",
			videoURL: "http://url",
			expect:   []string{"--extract-audio", "--audio-format", "mp3", "--format", "worstaudio", "--output", "/tmp/1", "http://url"},
		},
		{
			name:     "Audio best quality",
			format:   model.FormatAudio,
			quality:  model.QualityHigh,
			output:   "/tmp/1",
			videoURL: "http://url",
			expect:   []string{"--extract-audio", "--audio-format", "mp3", "--format", "bestaudio", "--output", "/tmp/1", "http://url"},
		},
		{
			name:     "Video unknown quality",
			format:   model.FormatVideo,
			output:   "/tmp/1",
			videoURL: "http://url",
			expect:   []string{"--format", "bestvideo[ext=mp4][vcodec^=avc1]+bestaudio[ext=m4a]/best[ext=mp4][vcodec^=avc1]/best[ext=mp4]/best", "--output", "/tmp/1", "http://url"},
		},
		{
			name:      "Video unknown quality with maxheight",
			format:    model.FormatVideo,
			maxHeight: 720,
			output:    "/tmp/1",
			videoURL:  "http://url",
			expect:    []string{"--format", "bestvideo[ext=mp4][vcodec^=avc1]+bestaudio[ext=m4a]/best[ext=mp4][vcodec^=avc1]/best[ext=mp4]/best", "--output", "/tmp/1", "http://url"},
		},
		{
			name:     "Video low quality",
			format:   model.FormatVideo,
			quality:  model.QualityLow,
			output:   "/tmp/2",
			videoURL: "http://url",
			expect:   []string{"--format", "worstvideo[ext=mp4][vcodec^=avc1]+worstaudio[ext=m4a]/worst[ext=mp4][vcodec^=avc1]/worst[ext=mp4]/worst", "--output", "/tmp/2", "http://url"},
		},
		{
			name:      "Video low quality with maxheight",
			format:    model.FormatVideo,
			quality:   model.QualityLow,
			maxHeight: 720,
			output:    "/tmp/2",
			videoURL:  "http://url",
			expect:    []string{"--format", "worstvideo[ext=mp4][vcodec^=avc1]+worstaudio[ext=m4a]/worst[ext=mp4][vcodec^=avc1]/worst[ext=mp4]/worst", "--output", "/tmp/2", "http://url"},
		},
		{
			name:     "Video high quality",
			format:   model.FormatVideo,
			quality:  model.QualityHigh,
			output:   "/tmp/2",
			videoURL: "http://url1",
			expect:   []string{"--format", "bestvideo[ext=mp4][vcodec^=avc1]+bestaudio[ext=m4a]/best[ext=mp4][vcodec^=avc1]/best[ext=mp4]/best", "--output", "/tmp/2", "http://url1"},
		},
		{
			name:      "Video high quality with maxheight",
			format:    model.FormatVideo,
			quality:   model.QualityHigh,
			maxHeight: 1024,
			output:    "/tmp/2",
			videoURL:  "http://url1",
			expect:    []string{"--format", "bestvideo[height<=1024][ext=mp4][vcodec^=avc1]+bestaudio[ext=m4a]/best[height<=1024][ext=mp4][vcodec^=avc1]/best[ext=mp4]/best", "--output", "/tmp/2", "http://url1"},
		},
		{
			name:     "Video high quality with custom youtube-dl arguments",
			format:   model.FormatVideo,
			quality:  model.QualityHigh,
			output:   "/tmp/2",
			videoURL: "http://url1",
			ytdlArgs: []string{"--write-sub", "--embed-subs", "--sub-lang", "en,en-US,en-GB"},
			expect:   []string{"--format", "bestvideo[ext=mp4][vcodec^=avc1]+bestaudio[ext=m4a]/best[ext=mp4][vcodec^=avc1]/best[ext=mp4]/best", "--write-sub", "--embed-subs", "--sub-lang", "en,en-US,en-GB", "--output", "/tmp/2", "http://url1"},
		},
		{
			name:         "Custom format",
			format:       model.FormatCustom,
			customFormat: feed.CustomFormat{YouTubeDLFormat: "bestaudio[ext=m4a]", Extension: "m4a"},
			quality:      model.QualityHigh,
			output:       "/tmp/2",
			videoURL:     "http://url1",
			expect:       []string{"--audio-format", "m4a", "--format", "bestaudio[ext=m4a]", "--output", "/tmp/2", "http://url1"},
		},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			result := buildArgs(&feed.Config{
				Format:        tst.format,
				Quality:       tst.quality,
				CustomFormat:  tst.customFormat,
				MaxHeight:     tst.maxHeight,
				YouTubeDLArgs: tst.ytdlArgs,
			}, &model.Episode{
				VideoURL: tst.videoURL,
			}, tst.output, false)

			assert.EqualValues(t, tst.expect, result)
		})
	}
}

func TestBuildArgsVerbose(t *testing.T) {
	result := buildArgs(&feed.Config{
		Format:  model.FormatAudio,
		Quality: model.QualityHigh,
	}, &model.Episode{
		VideoURL: "http://url",
	}, "/tmp/1", true)

	assert.Equal(t, []string{
		"--extract-audio", "--audio-format", "mp3", "--format", "bestaudio",
		"--newline",
		"--output", "/tmp/1", "http://url",
	}, result)
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KiB"},
		{1536, "1.50 KiB"},
		{1024 * 1024, "1.00 MiB"},
		{int64(1.5 * 1024 * 1024), "1.50 MiB"},
		{1024 * 1024 * 1024, "1.00 GiB"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, humanBytes(tc.n))
	}
}

func TestListRegularFiles(t *testing.T) {
	dir := t.TempDir()

	assert.Empty(t, listRegularFiles(dir), "empty dir should return empty map")

	want := map[string]int{"small.txt": 10, "big.bin": 1000, "mid.log": 100}
	for name, size := range want {
		path := filepath.Join(dir, name)
		assert.NoError(t, os.WriteFile(path, make([]byte, size), 0o644))
	}

	assert.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))

	got := listRegularFiles(dir)
	assert.Len(t, got, 3, "sub-directories should be skipped")
	for name, size := range want {
		assert.EqualValues(t, size, got[name], "size mismatch for %q", name)
	}
}

func TestScanLinesCR(t *testing.T) {
	input := "[info] starting\n" +
		"[download]   0.0% of 1MiB at 1MiB/s\r" +
		"[download]  50.0% of 1MiB at 1MiB/s\r" +
		"[download] 100.0% of 1MiB at 1MiB/s\n" +
		"[info] done"

	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(scanLinesCR)

	var tokens []string
	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}

	assert.NoError(t, scanner.Err())
	assert.Equal(t, []string{
		"[info] starting",
		"[download]   0.0% of 1MiB at 1MiB/s",
		"[download]  50.0% of 1MiB at 1MiB/s",
		"[download] 100.0% of 1MiB at 1MiB/s",
		"[info] done",
	}, tokens)
}
