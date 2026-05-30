package metrics

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"sealchat/model"
	"sealchat/service"
	"sealchat/service/storage"
)

const attachmentImageFullScanInterval = 7 * 24 * time.Hour

type fileStats struct {
	Count int64
	Bytes int64
}

type attachmentStorageStatsCache struct {
	mu              sync.Mutex
	fullScanAt      time.Time
	fullScanMonth   string
	priorMonthImage fileStats
}

var defaultAttachmentStorageStatsCache = newAttachmentStorageStatsCache()

func newAttachmentStorageStatsCache() *attachmentStorageStatsCache {
	return &attachmentStorageStatsCache{}
}

func LoadAttachmentStatusStats(now time.Time) (*model.AttachmentStatusStats, error) {
	base, err := model.CountAttachmentStatusStats()
	if err != nil {
		return nil, err
	}
	return defaultAttachmentStorageStatsCache.mergeActualStorageStats(now, service.GetStorageManager(), base)
}

func (c *attachmentStorageStatsCache) mergeActualStorageStats(
	now time.Time,
	manager *storage.Manager,
	base *model.AttachmentStatusStats,
) (*model.AttachmentStatusStats, error) {
	if base == nil {
		base = &model.AttachmentStatusStats{}
	}
	stats := *base
	if manager == nil {
		stats.TotalCount = stats.ImageCount + stats.FontCount
		stats.TotalBytes = stats.ImageBytes + stats.FontBytes
		return &stats, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()

	if manager.ActiveBackendForAttachment() == storage.BackendLocal {
		priorStats, err := c.loadPriorMonthImageStats(now, manager)
		if err != nil {
			return nil, err
		}
		currentStats, err := scanLocalObjectTree(manager, path.Join("attachments", now.Format("2006"), now.Format("01")))
		if err != nil {
			return nil, err
		}
		stats.ImageCount = priorStats.Count + currentStats.Count
		stats.ImageBytes = priorStats.Bytes + currentStats.Bytes
	}

	stats.TotalCount = stats.ImageCount + stats.FontCount
	stats.TotalBytes = stats.ImageBytes + stats.FontBytes
	return &stats, nil
}

func (c *attachmentStorageStatsCache) loadPriorMonthImageStats(now time.Time, manager *storage.Manager) (fileStats, error) {
	monthKey := now.Format("2006/01")

	c.mu.Lock()
	defer c.mu.Unlock()

	needFullScan := c.fullScanAt.IsZero() ||
		now.Sub(c.fullScanAt) >= attachmentImageFullScanInterval ||
		c.fullScanMonth != monthKey
	if !needFullScan {
		return c.priorMonthImage, nil
	}

	root, err := resolveLocalRoot(manager, "attachments")
	if err != nil {
		return fileStats{}, err
	}
	priorStats, err := scanAttachmentPriorMonths(root, monthKey)
	if err != nil {
		return fileStats{}, err
	}
	c.priorMonthImage = priorStats
	c.fullScanAt = now
	c.fullScanMonth = monthKey
	return c.priorMonthImage, nil
}

func resolveLocalRoot(manager *storage.Manager, prefix string) (string, error) {
	probePath, err := manager.ResolveLocalPath(path.Join(prefix, "__sealchat_probe__"))
	if err != nil {
		return "", err
	}
	return filepath.Dir(probePath), nil
}

func scanAttachmentPriorMonths(root string, currentMonthKey string) (fileStats, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fileStats{}, nil
		}
		return fileStats{}, err
	}
	if !info.IsDir() {
		return fileStats{}, nil
	}

	var stats fileStats
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(relative)
		if normalized == "" {
			return nil
		}
		if len(normalized) >= len(currentMonthKey)+1 && normalized[:len(currentMonthKey)+1] == currentMonthKey+"/" {
			return nil
		}
		stats.Count++
		stats.Bytes += info.Size()
		return nil
	})
	if err != nil {
		return fileStats{}, err
	}
	return stats, nil
}

func scanLocalObjectTree(manager *storage.Manager, objectKey string) (fileStats, error) {
	root, err := manager.ResolveLocalPath(objectKey)
	if err != nil {
		return fileStats{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fileStats{}, nil
		}
		return fileStats{}, err
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			return fileStats{}, nil
		}
		return fileStats{Count: 1, Bytes: info.Size()}, nil
	}

	var stats fileStats
	err = filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		stats.Count++
		stats.Bytes += info.Size()
		return nil
	})
	if err != nil {
		return fileStats{}, err
	}
	return stats, nil
}
