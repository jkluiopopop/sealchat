package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sealchat/model"
	"sealchat/service"
	"sealchat/service/storage"
	"sealchat/utils"
)

func PlatformFontListPublicHandler(c *fiber.Ctx) error {
	items, err := service.PlatformFontListPublic()
	if err != nil {
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "加载平台字体列表失败")
	}
	return c.JSON(fiber.Map{
		"items": items,
	})
}

func PlatformFontMetaHandler(c *fiber.Ctx) error {
	item, err := service.PlatformFontGet(c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrPlatformFontNotFound) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "平台字体不存在")
		}
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "加载平台字体详情失败")
	}
	if item.Status != model.PlatformFontStatusReady {
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体不可用")
	}
	return c.JSON(item)
}

func PlatformFontFileHandler(c *fiber.Ctx) error {
	item, err := service.PlatformFontGet(c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrPlatformFontNotFound) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "平台字体不存在")
		}
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "加载平台字体失败")
	}
	if item.Status != model.PlatformFontStatusReady || strings.TrimSpace(item.OriginalObjectKey) == "" {
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体不可用")
	}
	if item.OriginalStorageType == model.StorageFontS3 {
		if redirected := redirectPlatformFontToRemote(c, item.OriginalStorageType, item.OriginalObjectKey); redirected {
			return nil
		}
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体文件不存在")
	}
	path, err := service.ResolveLocalPlatformFontPath(item.OriginalObjectKey)
	if err != nil {
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "解析平台字体路径失败")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "平台字体文件不存在")
		}
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "读取平台字体文件失败")
	}
	setPlatformFontResponseHeaders(c, item.SourceMimeType, item.SourceFileName)
	return c.SendFile(path)
}

func PlatformFontSubsetManifestHandler(c *fiber.Ctx) error {
	item, err := service.PlatformFontGet(c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrPlatformFontNotFound) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "平台字体不存在")
		}
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "加载平台字体失败")
	}
	if item.Status != model.PlatformFontStatusReady || item.DeliveryMode != model.PlatformFontDeliverySubset || strings.TrimSpace(item.ManifestObjectKey) == "" {
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体分片清单不存在")
	}
	manifest, err := loadPlatformFontSubsetManifest(item)
	if err != nil {
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "读取平台字体分片清单失败")
	}
	webURL := ""
	if cfg := utils.GetConfig(); cfg != nil {
		webURL = cfg.WebUrl
	}
	return c.JSON(normalizePlatformFontSubsetManifestForResponse(webURL, item.ID, manifest))
}

func PlatformFontSubsetFileHandler(c *fiber.Ctx) error {
	item, err := service.PlatformFontGet(c.Params("id"))
	if err != nil {
		if errors.Is(err, service.ErrPlatformFontNotFound) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "平台字体不存在")
		}
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "加载平台字体失败")
	}
	if item.Status != model.PlatformFontStatusReady || item.DeliveryMode != model.PlatformFontDeliverySubset || strings.TrimSpace(item.SubsetObjectKey) == "" {
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体分片不存在")
	}
	name := strings.TrimSpace(c.Params("*"))
	if name == "" {
		name = strings.TrimSpace(c.Params("name"))
	}
	if name == "" || strings.Contains(name, "..") {
		return wrapErrorStatus(c, fiber.StatusBadRequest, nil, "无效的字体分片名称")
	}
	objectKey := storage.BuildPlatformFontSubsetObjectKey(item.ID, name)
	return sendPlatformFontObject(c, item.SubsetStorageType, objectKey, "", name)
}

func sendPlatformFontObject(
	c *fiber.Ctx,
	storageType model.StorageType,
	objectKey string,
	contentType string,
	filename string,
) error {
	if strings.TrimSpace(objectKey) == "" {
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体文件不存在")
	}
	if storageType == model.StorageFontS3 {
		if redirected := redirectPlatformFontToRemote(c, storageType, objectKey); redirected {
			return nil
		}
		return wrapErrorStatus(c, fiber.StatusNotFound, nil, "平台字体文件不存在")
	}
	path, err := service.ResolveLocalPlatformFontPath(objectKey)
	if err != nil {
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "解析平台字体路径失败")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "平台字体文件不存在")
		}
		return wrapErrorStatus(c, fiber.StatusInternalServerError, err, "读取平台字体文件失败")
	}
	setPlatformFontResponseHeaders(c, contentType, filename)
	return c.SendFile(path)
}

func redirectPlatformFontToRemote(c *fiber.Ctx, storageType model.StorageType, objectKey string) bool {
	manager := service.GetStorageManager()
	if manager == nil {
		return false
	}
	target := manager.ResolveAttachmentExportURL(context.Background(), convertPlatformFontStorageToBackend(storageType), objectKey)
	if strings.TrimSpace(target) == "" {
		return false
	}
	_ = c.Redirect(target, fiber.StatusTemporaryRedirect)
	return true
}

func convertPlatformFontStorageToBackend(storageType model.StorageType) storage.BackendType {
	if storageType == model.StorageFontS3 {
		return storage.BackendS3
	}
	return storage.BackendLocal
}

func setPlatformFontResponseHeaders(c *fiber.Ctx, contentType string, filename string) {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct == "" || ct == "application/octet-stream" {
		ct = "application/octet-stream"
	}
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("Content-Type", ct)
	if strings.HasPrefix(ct, "font/") || ct == "application/font-sfnt" || ct == "application/x-font-ttf" {
		return
	}
	name := sanitizePlatformFontFilename(filename)
	if name == "" {
		name = "platform-font.bin"
	}
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
}

func sanitizePlatformFontFilename(value string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '"', '\\', '\r', '\n':
			return -1
		default:
			return r
		}
	}, name)
}

func loadPlatformFontSubsetManifest(item *model.PlatformFontAsset) (*service.PlatformFontSubsetManifestData, error) {
	if item == nil {
		return nil, errors.New("platform font asset is nil")
	}
	if strings.TrimSpace(item.ManifestObjectKey) == "" {
		return nil, errors.New("platform font manifest object key is empty")
	}
	if item.ManifestStorageType == model.StorageFontS3 {
		manager := service.GetStorageManager()
		if manager == nil {
			return nil, errors.New("storage manager not initialized")
		}
		target := manager.ResolveAttachmentExportURL(context.Background(), convertPlatformFontStorageToBackend(item.ManifestStorageType), item.ManifestObjectKey)
		if strings.TrimSpace(target) == "" {
			return nil, errors.New("platform font manifest remote url is empty")
		}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected manifest status: %d", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return decodePlatformFontSubsetManifest(body)
	}
	localPath, err := service.ResolveLocalPlatformFontPath(item.ManifestObjectKey)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	return decodePlatformFontSubsetManifest(body)
}

func decodePlatformFontSubsetManifest(body []byte) (*service.PlatformFontSubsetManifestData, error) {
	var manifest service.PlatformFontSubsetManifestData
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func normalizePlatformFontSubsetManifestForResponse(webURL string, fontID string, manifest *service.PlatformFontSubsetManifestData) *service.PlatformFontSubsetManifestData {
	return service.NormalizePlatformFontSubsetManifestURLs(fontID, webURL, manifest)
}
