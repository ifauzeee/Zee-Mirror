package service

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"

	"zee-mirror/internal/config"
	"zee-mirror/internal/domain"
	"zee-mirror/internal/downloader"
	"zee-mirror/internal/uploader"
	"zee-mirror/pkg/utils"
)

func newTestTask(id, gid string, status TaskStatus, createdAt time.Time) *Task {
	return &Task{
		Task: domain.Task{
			ID:        id,
			GID:       gid,
			Status:    status,
			Type:      TypeMirror,
			CreatedAt: createdAt,
		},
	}
}

func TestMaskToken(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "****"},
		{"12345678", "****"},
		{"123456789", "1234...6789"},
		{"1234567890123456:ABCdefGHI", "1234...fGHI"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, maskToken(c.in), "input %q", c.in)
	}
}

func TestIsMediaMessage(t *testing.T) {
	assert.False(t, IsMediaMessage(nil))
	assert.False(t, IsMediaMessage(&tgbotapi.Message{}))

	doc := &tgbotapi.Message{Document: &tgbotapi.Document{}}
	assert.True(t, IsMediaMessage(doc))

	photo := &tgbotapi.Message{Photo: []tgbotapi.PhotoSize{{FileID: "x"}}}
	assert.True(t, IsMediaMessage(photo))
}

func TestCalculateDuration(t *testing.T) {
	now := time.Now()

	snap := domain.TaskSnapshot{StartedAt: now.Add(-2 * time.Hour), CompletedAt: now}
	assert.Equal(t, 2*time.Hour, calculateDuration(snap))

	snap = domain.TaskSnapshot{CreatedAt: now.Add(-5 * time.Hour), CompletedAt: now}
	assert.Equal(t, 5*time.Hour, calculateDuration(snap))
}

func TestDetermineSizeString(t *testing.T) {
	snap := domain.TaskSnapshot{TotalSize: 1500}
	assert.Equal(t, utils.FormatBytes(1500), determineSizeString(snap))

	snap = domain.TaskSnapshot{DownloadedSize: 2048}
	assert.Equal(t, utils.FormatBytes(2048), determineSizeString(snap))

	snap = domain.TaskSnapshot{}
	assert.Equal(t, UnknownSize, determineSizeString(snap))

	snap = domain.TaskSnapshot{LocalPath: filepath.Join(t.TempDir(), "missing.bin"), TotalSize: 300}
	assert.Equal(t, utils.FormatBytes(300), determineSizeString(snap))
}

func TestStripANSI(t *testing.T) {
	assert.Equal(t, "ERR plain", stripANSI("\x1b[31mERR\x1b[0m plain"))
	assert.Equal(t, "clean", stripANSI("clean"))
}

func TestParseTorrentOutput_PipeFormat(t *testing.T) {
	out := "idx|path|length\n===+===+===\n0 | dir/fileA.mkv | 100 | true\n"
	files := parseTorrentOutput(out)
	assert.Len(t, files, 1)
	assert.Equal(t, domain.TorrentFile{Index: 0, Name: "fileA.mkv", Path: "dir/fileA.mkv", Size: 100}, files[0])
}

func TestParseTorrentOutput_SpaceFormat(t *testing.T) {
	out := "1 fileB.mp4 200 false\n"
	files := parseTorrentOutput(out)
	assert.Len(t, files, 1)
	assert.Equal(t, domain.TorrentFile{Index: 1, Name: "fileB.mp4", Path: "fileB.mp4", Size: 200}, files[0])
}

func TestParseTorrentOutput_HeaderPlusSizeLines(t *testing.T) {
	out := "2|dir/sub/fileC.zip\n   |(1,300)\n"
	files := parseTorrentOutput(out)
	assert.Len(t, files, 1)
	assert.Equal(t, domain.TorrentFile{Index: 2, Name: "fileC.zip", Path: "dir/sub/fileC.zip", Size: 1300}, files[0])
}

func TestParseTorrentOutput_SkipsNoise(t *testing.T) {
	files := parseTorrentOutput("===+===+===\n---\nidx|path\n\n")
	assert.Empty(t, files)
}

func TestDetermineTorrentName(t *testing.T) {
	bs := &BotService{}

	same := []domain.TorrentFile{
		{Name: "a.mkv", Path: "Movie/a.mkv"},
		{Name: "b.mkv", Path: "Movie/b.mkv"},
	}
	assert.Equal(t, "Movie", bs.determineTorrentName(same))

	dotPrefix := []domain.TorrentFile{
		{Name: "s1e1.mkv", Path: "./Show/s1e1.mkv"},
		{Name: "s1e2.mkv", Path: "./Show/s1e2.mkv"},
	}
	assert.Equal(t, "Show", bs.determineTorrentName(dotPrefix))

	diff := []domain.TorrentFile{
		{Name: "a", Path: "X/a"},
		{Name: "b", Path: "Y/b"},
	}
	assert.Empty(t, bs.determineTorrentName(diff))

	single := []domain.TorrentFile{{Name: "x.iso", Path: "x.iso"}}
	assert.Equal(t, "x.iso", bs.determineTorrentName(single))
}

func TestParseBatchArguments(t *testing.T) {
	urls := "http://example.com/1.zip\nhttp://example.com/2.zip"
	opts := parseBatchArguments(urls)
	assert.Equal(t, []string{"http://example.com/1.zip", "http://example.com/2.zip"}, opts.URLs)
	assert.Equal(t, 5, opts.Priority)
	assert.False(t, opts.ZipAll)

	opts = parseBatchArguments("-name My Batch\nhttp://example.com/a")
	assert.Equal(t, "My Batch", opts.Name)

	opts = parseBatchArguments("-n short\nhttp://example.com/a")
	assert.Equal(t, "short", opts.Name)

	opts = parseBatchArguments("-zip\n-z\nhttp://example.com/a")
	assert.True(t, opts.ZipAll)

	opts = parseBatchArguments("-p s3cret\nhttp://example.com/a")
	assert.Equal(t, "s3cret", opts.Password)

	opts = parseBatchArguments("-priority 3\nhttp://example.com/a")
	assert.Equal(t, 3, opts.Priority)

	opts = parseBatchArguments("-priority abc")
	assert.Equal(t, 5, opts.Priority)

	opts = parseBatchArguments("-priority 99")
	assert.Equal(t, 10, opts.Priority)

	opts = parseBatchArguments("-priority 0")
	assert.Equal(t, 1, opts.Priority)

	opts = parseBatchArguments("not a url\n\n")
	assert.Empty(t, opts.URLs)
}

func TestFormatDriveFileList(t *testing.T) {
	bs := &BotService{}

	text := bs.FormatDriveFileList("/root", nil)
	assert.Contains(t, text, "/root")
	assert.Contains(t, text, "kosong")

	files := []DriveFile{
		{Name: "Movies", IsDir: true},
		{Name: "file1.zip", Size: 1024},
		{Name: "file2.zip", Size: 2048},
	}
	text = bs.FormatDriveFileList("/data", files)
	assert.Contains(t, text, "Movies")
	assert.Contains(t, text, "file1.zip")
	assert.Contains(t, text, "Folders: `1`")
	assert.Contains(t, text, "Files: `2`")
}

func TestGetTaskByGID(t *testing.T) {
	tm := &TaskManager{Tasks: make(map[string]*Task)}
	task := newTestTask("id1", "gid1", StatusDownloading, time.Now())
	tm.Tasks["id1"] = task

	assert.Same(t, task, tm.GetTaskByGID("gid1"))
	assert.Nil(t, tm.GetTaskByGID("missing"))
}

func TestGetActiveTasks_FiltersAndSorts(t *testing.T) {
	base := time.Now()
	tm := &TaskManager{Tasks: make(map[string]*Task)}

	old := newTestTask("t1", "g1", StatusDownloading, base.Add(2*time.Hour))
	mid := newTestTask("t2", "g2", StatusQueued, base.Add(1*time.Hour))
	done := newTestTask("t3", "g3", StatusCompleted, base)

	tm.Tasks[old.ID] = old
	tm.Tasks[mid.ID] = mid
	tm.Tasks[done.ID] = done

	active := tm.GetActiveTasks()
	assert.Len(t, active, 2)
	assert.Equal(t, "t2", active[0].ID)
	assert.Equal(t, "t1", active[1].ID)
}

func TestIsShuttingDown(t *testing.T) {
	tm := &TaskManager{ShutdownChan: make(chan struct{})}
	assert.False(t, tm.IsShuttingDown())

	close(tm.ShutdownChan)
	assert.True(t, tm.IsShuttingDown())
}

func TestSetProgress(t *testing.T) {
	task := newTestTask("t1", "g1", StatusDownloading, time.Now())
	task.SetProgress(42.5)
	assert.InDelta(t, 42.5, task.Progress, 0.001)
}

func TestCompleteTelegramUpload(t *testing.T) {
	task := newTestTask("t1", "g1", StatusUploading, time.Now())
	task.CompleteTelegramUpload(42, 999, "fid", "fpath")

	assert.Equal(t, 42, task.ResultMessageID)
	assert.InDelta(t, 100, task.Progress, 0.001)
	assert.Equal(t, int64(999), task.UploadedSize)
	assert.Equal(t, "telegram", task.RemotePath)
	assert.Equal(t, "fid", task.TelegramFileID)
	assert.Equal(t, "fpath", task.TelegramFilePath)
}

func TestUpdateFromProgressUpdate(t *testing.T) {
	task := newTestTask("t1", "g1", StatusDownloading, time.Now())
	up := downloader.ProgressUpdate{
		FileName:    "new.mkv",
		Downloaded:  100,
		Total:       200,
		Speed:       50,
		Progress:    50,
		Connections: 8,
		ETA:         10 * time.Second,
	}
	task.UpdateFromProgressUpdate(up)

	assert.Equal(t, "new.mkv", task.FileName)
	assert.Equal(t, int64(100), task.DownloadedSize)
	assert.Equal(t, int64(200), task.TotalSize)
	assert.Equal(t, int64(50), task.Speed)
	assert.InDelta(t, 50, task.Progress, 0.001)
	assert.Equal(t, 8, task.Connections)
	assert.Equal(t, 10*time.Second, task.ETA)

	// Message resets speed/ETA.
	task.UpdateFromProgressUpdate(downloader.ProgressUpdate{Message: "processing"})
	assert.Equal(t, "processing", task.ProcessingMessage)
	assert.Zero(t, task.Speed)
	assert.Zero(t, task.ETA)

	// Zero fields are ignored except Progress which is always overwritten.
	task.UpdateFromProgressUpdate(downloader.ProgressUpdate{})
	assert.InDelta(t, 0, task.Progress, 0.001)
	assert.Equal(t, int64(100), task.DownloadedSize)
}

func TestUpdateFromUploadProgress(t *testing.T) {
	task := newTestTask("t1", "g1", StatusUploading, time.Now())
	up := uploader.ProgressUpdate{UploadedSize: 100, TotalSize: 200, Progress: 50, Speed: 10, ETA: 5 * time.Second}
	task.UpdateFromUploadProgress(up)

	assert.Equal(t, int64(100), task.UploadedSize)
	assert.Equal(t, int64(200), task.TotalSize)
	assert.InDelta(t, 50, task.Progress, 0.001)
	assert.Equal(t, int64(10), task.Speed)
	assert.Equal(t, 5*time.Second, task.ETA)

	task.UpdateFromUploadProgress(uploader.ProgressUpdate{Error: "boom"})
	assert.Equal(t, "boom", task.Error)

	// Zeros ignored.
	task.UpdateFromUploadProgress(uploader.ProgressUpdate{})
	assert.Equal(t, int64(100), task.UploadedSize)
}

func TestCalculateSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	sum, err := calculateSHA256(path)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(content)), sum)

	_, err = calculateSHA256(filepath.Join(t.TempDir(), "missing"))
	assert.Error(t, err)
}

func TestSetStatus(t *testing.T) {
	task := newTestTask("t1", "g1", StatusQueued, time.Now())
	task.SetStatus(StatusDownloading)
	assert.Equal(t, StatusDownloading, task.Status)
	assert.Equal(t, 16, task.Connections)

	task.SetStatus(StatusCompleted)
	assert.False(t, task.CompletedAt.IsZero())

	// Cancelled is terminal: later status changes are ignored.
	cancelled := newTestTask("t2", "g2", StatusCancelled, time.Now())
	cancelled.SetStatus(StatusDownloading)
	assert.Equal(t, StatusCancelled, cancelled.Status)
}

func TestSetError(t *testing.T) {
	task := newTestTask("t1", "g1", StatusDownloading, time.Now())
	task.SetError("kaput")

	assert.Equal(t, "kaput", task.Error)
	assert.Equal(t, StatusFailed, task.Status)
	assert.False(t, task.CompletedAt.IsZero())
}

func TestCancelTask(t *testing.T) {
	tm := &TaskManager{Tasks: make(map[string]*Task)}

	active := newTestTask("t1", "g1", StatusDownloading, time.Now())
	tm.Tasks[active.ID] = active
	assert.True(t, tm.CancelTask("t1"))
	assert.NotContains(t, tm.Tasks, "t1")
	assert.Equal(t, StatusCancelled, active.Status)
	assert.False(t, active.CompletedAt.IsZero())

	assert.False(t, tm.CancelTask("missing"))

	done := newTestTask("t3", "g3", StatusCompleted, time.Now())
	tm.Tasks[done.ID] = done
	assert.False(t, tm.CancelTask("t3"))
	assert.NotContains(t, tm.Tasks, "t3") // removed even if not cancellable
}

func TestCancelAllTasks(t *testing.T) {
	tm := &TaskManager{Tasks: make(map[string]*Task)}
	for _, tk := range []*Task{
		newTestTask("t1", "g1", StatusDownloading, time.Now()),
		newTestTask("t2", "g2", StatusQueued, time.Now()),
		newTestTask("t3", "g3", StatusCompleted, time.Now()),
	} {
		tm.Tasks[tk.ID] = tk
	}

	assert.Equal(t, 2, tm.CancelAllTasks())
	assert.Empty(t, tm.Tasks)
}

func TestGetFileFromMessage(t *testing.T) {
	bs := &BotService{}

	fileID, fileName := bs.GetFileFromMessage(&tgbotapi.Message{})
	assert.Empty(t, fileID)
	assert.Empty(t, fileName)

	reply := &tgbotapi.Message{Video: &tgbotapi.Video{FileID: "v1", FileName: "clip.mp4"}}
	fileID, fileName = bs.GetFileFromMessage(&tgbotapi.Message{ReplyToMessage: reply})
	assert.Equal(t, "v1", fileID)
	assert.Equal(t, "clip.mp4", fileName)

	reply = &tgbotapi.Message{Voice: &tgbotapi.Voice{FileID: "vo1"}}
	fileID, fileName = bs.GetFileFromMessage(&tgbotapi.Message{ReplyToMessage: reply})
	assert.Equal(t, "vo1", fileID)
	assert.Empty(t, fileName)
}

func TestExtractFileFromReply(t *testing.T) {
	bs := &BotService{}

	fileID, fileName, size := bs.ExtractFileFromReply(&tgbotapi.Message{})
	assert.Empty(t, fileID)
	assert.Empty(t, fileName)
	assert.Zero(t, size)

	doc := &tgbotapi.Message{Document: &tgbotapi.Document{FileID: "d1", FileName: "a.zip", FileSize: 123}}
	fileID, fileName, size = bs.ExtractFileFromReply(doc)
	assert.Equal(t, "d1", fileID)
	assert.Equal(t, "a.zip", fileName)
	assert.Equal(t, int64(123), size)

	voice := &tgbotapi.Message{Voice: &tgbotapi.Voice{FileID: "vo1", FileSize: 456}}
	fileID, fileName, size = bs.ExtractFileFromReply(voice)
	assert.Equal(t, "vo1", fileID)
	assert.Contains(t, fileName, ".ogg")
	assert.Equal(t, int64(456), size)
}

func TestGetFileLink(t *testing.T) {
	bs := &BotService{Config: &config.Config{BotToken: "TOK"}}

	link := bs.GetFileLink(tgbotapi.File{FilePath: "docs/a.txt"}, true)
	assert.Contains(t, link, "TOK")
	assert.Contains(t, link, "docs/a.txt")

	bs.Config.TelegramAPI = "http://localhost:8080/bot%s/%s"
	link = bs.GetFileLink(tgbotapi.File{FilePath: "/docs/a.txt"}, false)
	assert.Equal(t, "http://localhost:8080/file/botTOK/docs/a.txt", link)

	bs.Config.TelegramAPI = "http://localhost:8080"
	link = bs.GetFileLink(tgbotapi.File{FilePath: "/docs/a.txt"}, false)
	assert.Equal(t, "http://localhost:8080/file/botTOK/docs/a.txt", link)
}
