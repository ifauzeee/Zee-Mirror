package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDownloadedFile(t *testing.T) {
	t.Run("SingleFile", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("data"), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("MultipleFiles_ReturnsLargest", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "small.mp4"), []byte("small"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "large.mp4"), make([]byte, 1000), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "large.mp4"), result)
	})

	t.Run("SkipsAria2Files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4.aria2"), []byte("data"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("data"), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("SkipsPartFiles", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4.part"), []byte("data"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("data"), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("SkipsTempFiles", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "download.temp"), []byte("data"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("data"), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("SkipsTorrentFiles", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.torrent"), []byte("data"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("data"), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("QualityAudio_SelectsAudioFile", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), make([]byte, 500), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "audio.mp3"), make([]byte, 100), 0600))

		result := findDownloadedFile(dir, "audio")
		assert.Equal(t, filepath.Join(dir, "audio.mp3"), result)
	})

	t.Run("QualityVideo_SelectsVideoFile", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "audio.mp3"), make([]byte, 500), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), make([]byte, 100), 0600))

		result := findDownloadedFile(dir, "720p")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("QualityAudio_FallbackToLargest", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), make([]byte, 500), 0600))

		result := findDownloadedFile(dir, "audio")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("EmptyDir", func(t *testing.T) {
		dir := t.TempDir()
		result := findDownloadedFile(dir, "")
		assert.Empty(t, result)
	})

	t.Run("NonExistentDir", func(t *testing.T) {
		result := findDownloadedFile("/nonexistent/path", "")
		assert.Empty(t, result)
	})

	t.Run("SkipsSubdirectories", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mp4"), []byte("data"), 0600))

		result := findDownloadedFile(dir, "")
		assert.Equal(t, filepath.Join(dir, "video.mp4"), result)
	})

	t.Run("MultipleAudioFormats", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "song.flac"), make([]byte, 200), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "song.opus"), make([]byte, 100), 0600))

		result := findDownloadedFile(dir, "audio")
		assert.Equal(t, filepath.Join(dir, "song.flac"), result)
	})

	t.Run("MultipleVideoFormats", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.mkv"), make([]byte, 200), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "video.webm"), make([]byte, 100), 0600))

		result := findDownloadedFile(dir, "1080p")
		assert.Equal(t, filepath.Join(dir, "video.mkv"), result)
	})
}

func TestGetLargest(t *testing.T) {
	t.Run("SinglePath", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0600))

		result := getLargest([]string{path})
		assert.Equal(t, path, result)
	})

	t.Run("MultiplePaths", func(t *testing.T) {
		dir := t.TempDir()
		small := filepath.Join(dir, "small.txt")
		large := filepath.Join(dir, "large.txt")
		require.NoError(t, os.WriteFile(small, []byte("hi"), 0600))
		require.NoError(t, os.WriteFile(large, make([]byte, 100), 0600))

		result := getLargest([]string{small, large})
		assert.Equal(t, large, result)
	})

	t.Run("EmptyPaths", func(t *testing.T) {
		result := getLargest([]string{})
		assert.Empty(t, result)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		result := getLargest([]string{"/nonexistent/file.txt"})
		assert.Empty(t, result)
	})

	t.Run("MixedExistentAndNonExistent", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(existing, []byte("data"), 0600))

		result := getLargest([]string{"/nonexistent/file.txt", existing})
		assert.Equal(t, existing, result)
	})
}
