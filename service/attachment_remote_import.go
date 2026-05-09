package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"sealchat/model"
)

var (
	ErrRemoteAttachmentURLInvalid = errors.New("远端附件 URL 无效")
	ErrRemoteAttachmentTooLarge   = errors.New("远端附件大小超过限制")
)

type RemoteAttachmentImportInput struct {
	URL          string
	Filename     string
	ContentType  string
	UserID       string
	ChannelID    string
	MaxSizeBytes int64
	HTTPClient   *http.Client
}

func ImportAttachmentFromURL(input RemoteAttachmentImportInput) (*model.AttachmentModel, error) {
	parsed, err := normalizeRemoteAttachmentURL(input.URL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, errors.New("缺少用户 ID")
	}
	if GetStorageManager() == nil {
		return nil, errors.New("存储服务未初始化")
	}

	maxSize := input.MaxSizeBytes
	if maxSize <= 0 {
		maxSize = 20 * 1024 * 1024
	}
	client := input.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建下载请求失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载远端附件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("下载远端附件失败: HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxSize && resp.ContentLength > 0 {
		return nil, ErrRemoteAttachmentTooLarge
	}

	tempFile, err := os.CreateTemp("", "sealchat-remote-attachment-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	hasher := sha256.New()
	limited := io.LimitReader(resp.Body, maxSize+1)
	buffer := make([]byte, 32*1024)
	headerBuf := make([]byte, 0, 512)
	var total int64
	for {
		n, readErr := limited.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if _, err := hasher.Write(chunk); err != nil {
				return nil, fmt.Errorf("计算哈希失败: %w", err)
			}
			if _, err := tempFile.Write(chunk); err != nil {
				return nil, fmt.Errorf("写入临时文件失败: %w", err)
			}
			if len(headerBuf) < 512 {
				remain := 512 - len(headerBuf)
				if remain > n {
					remain = n
				}
				headerBuf = append(headerBuf, chunk[:remain]...)
			}
			total += int64(n)
			if total > maxSize {
				return nil, ErrRemoteAttachmentTooLarge
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("读取远端附件失败: %w", readErr)
		}
	}
	if total <= 0 {
		return nil, errors.New("远端附件为空")
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("关闭临时文件失败: %w", err)
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = strings.TrimSpace(resp.Header.Get("Content-Type"))
	}
	if contentType == "" {
		contentType = http.DetectContentType(headerBuf)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	filename := strings.TrimSpace(input.Filename)
	if filename == "" {
		if base := path.Base(parsed.Path); base != "" && base != "." && base != "/" {
			filename = base
		}
	}
	if filename == "" {
		filename = "remote-attachment"
	}

	hashBytes := hasher.Sum(nil)
	location, err := PersistAttachmentFile(hashBytes, total, tempPath, contentType)
	if err != nil {
		return nil, err
	}
	_, item := model.AttachmentCreate(&model.AttachmentModel{
		Filename:    filename,
		Size:        total,
		Hash:        hashBytes,
		MimeType:    contentType,
		UserID:      strings.TrimSpace(input.UserID),
		ChannelID:   strings.TrimSpace(input.ChannelID),
		StorageType: location.StorageType,
		ObjectKey:   location.ObjectKey,
		ExternalURL: location.ExternalURL,
	})
	return item, nil
}

func normalizeRemoteAttachmentURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ErrRemoteAttachmentURLInvalid
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, ErrRemoteAttachmentURLInvalid
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return nil, ErrRemoteAttachmentURLInvalid
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, ErrRemoteAttachmentURLInvalid
	}
	return parsed, nil
}
