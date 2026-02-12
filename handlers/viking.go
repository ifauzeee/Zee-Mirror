package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"zee-mirror/pkg/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	VikingAPIUrl = "https://vikingfile.com/api"
)

type VikingUploadInitResponse struct {
	UploadID    string   `json:"uploadId"`
	Key         string   `json:"key"`
	URLs        []string `json:"urls"`
	PartSize    int64    `json:"partSize"`
	NumberParts int      `json:"numberParts"`
}

type VikingCompleteResponse struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Hash string `json:"hash"`
	URL  string `json:"url"`
}

type VikingErrorResponse struct {
	Error string `json:"error"`
}

func (s *BotService) HandleViking(message *tgbotapi.Message, args string) {
	if err := s.CheckQuota(message.From.ID); err != nil {
		s.reply(message, GetErrorMessage("QUOTA EXCEEDED", err.Error()))
		return
	}

	url, zip, unzip, password, quality, name, _, _ := utils.ParseFlags(args)
	var fileName string

	if name != "" {
		fileName = name
	}

	if message.ReplyToMessage != nil {
		fileID, _ := s.extractFileFromReply(message.ReplyToMessage)
		if fileID != "" {
			s.reply(message, "❌ *Error*\n\nReply file untuk Viking belum disupport saat ini (hanya URL).")
			return
		}
	}

	if url != "" {
		if fileName == "" {
			fileName = utils.GetFileNameFromURL(url)
		}

		replyID := 0
		if message.ReplyToMessage != nil {
			replyID = message.ReplyToMessage.MessageID
		}

		task, err := s.TaskManager.CreateTask(TypeViking, url, fileName, message.Chat.ID, message.MessageID, replyID, message.From.ID, zip, unzip, password, quality, 0, "", false)
		if err != nil {
			s.handleCreateTaskError(message.Chat.ID, message.MessageID, err)
			return
		}
		s.UpdateSharedDashboard(message.Chat.ID, true)
		s.handleAutoDelete(task)
		slog.Info("Viking task created", "taskID", task.ID, "url", url)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "❌ *Error*\n\nBerikan URL untuk di-upload ke ViKiNG FiLE.")
	msg.ParseMode = MarkdownV2
	_, _ = s.Bot.Send(msg)
}

func (s *BotService) UploadToViking(task *Task) error {
	task.SetStatus(StatusUploading)
	task.Mu.Lock()
	task.Progress = 0
	task.UploadedSize = 0
	task.Mu.Unlock()
	s.updateTaskStatus(task)

	filePath := task.LocalPath
	if filePath == "" {
		return fmt.Errorf("no file to upload")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	fileSize := info.Size()

	slog.Info("Initiating Viking upload", "taskID", task.ID, "size", fileSize)
	initResp, err := s.vikingGetUploadURL(fileSize)
	if err != nil {
		return fmt.Errorf("failed to get upload url: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	partsETags := make([]string, initResp.NumberParts)

	var wg sync.WaitGroup
	errChan := make(chan error, initResp.NumberParts)
	sem := make(chan struct{}, 5)

	var uploadedBytes int64
	var mu sync.Mutex

	for i, partURL := range initResp.URLs {
		partNum := i

		wg.Add(1)
		sem <- struct{}{}

		go func(pNum int, pURL string) {
			defer wg.Done()
			defer func() { <-sem }()

			offset := int64(pNum) * initResp.PartSize
			size := initResp.PartSize
			if offset+size > fileSize {
				size = fileSize - offset
			}

			sectionReader := io.NewSectionReader(file, offset, size)

			etag, errUpload := s.vikingUploadPart(pURL, sectionReader, size)
			if errUpload != nil {
				errChan <- fmt.Errorf("part %d failed: %v", pNum+1, errUpload)
				return
			}
			partsETags[pNum] = etag

			mu.Lock()
			uploadedBytes += size
			task.Mu.Lock()
			task.UploadedSize = uploadedBytes
			task.Progress = float64(uploadedBytes) / float64(fileSize) * 100
			task.Mu.Unlock()
			mu.Unlock()

			if pNum%2 == 0 {
				s.updateTaskStatus(task)
			}
		}(partNum, partURL)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}

	slog.Info("Completing Viking upload", "taskID", task.ID)

	completeResp, err := s.vikingCompleteUpload(initResp.Key, initResp.UploadID, partsETags, task.FileName, s.Config.VikingUserHash)
	if err != nil {
		return fmt.Errorf("failed to complete upload: %v", err)
	}

	task.Mu.Lock()
	task.RemoteURL = completeResp.URL
	task.RemotePath = "viking://" + completeResp.Hash
	task.Progress = 100
	task.Mu.Unlock()

	return nil
}

func (s *BotService) vikingGetUploadURL(size int64) (*VikingUploadInitResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("size", fmt.Sprintf("%d", size)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", VikingAPIUrl+"/get-upload-url", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var res VikingUploadInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *BotService) vikingUploadPart(url string, reader io.Reader, size int64) (string, error) {
	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return "", err
	}
	req.ContentLength = size

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		etag = resp.Header.Get("Etag")
	}

	etag = strings.Trim(etag, "\"")

	return etag, nil
}

func (s *BotService) vikingCompleteUpload(key, uploadID string, parts []string, name, userHash string) (*VikingCompleteResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("key", key); err != nil {
		return nil, err
	}
	if err := writer.WriteField("uploadId", uploadID); err != nil {
		return nil, err
	}
	if err := writer.WriteField("name", name); err != nil {
		return nil, err
	}
	if userHash != "" {
		if err := writer.WriteField("user", userHash); err != nil {
			return nil, err
		}
	} else {
		if err := writer.WriteField("user", ""); err != nil {
			return nil, err
		}
	}

	for i, etag := range parts {
		partNum := i + 1
		if err := writer.WriteField(fmt.Sprintf("parts[%d][PartNumber]", i), fmt.Sprintf("%d", partNum)); err != nil {
			return nil, err
		}
		if err := writer.WriteField(fmt.Sprintf("parts[%d][ETag]", i), etag); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", VikingAPIUrl+"/complete-upload", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 1 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(responseBody))
	}

	var res VikingCompleteResponse
	if err := json.Unmarshal(responseBody, &res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v | body: %s", err, string(responseBody))
	}

	return &res, nil
}
