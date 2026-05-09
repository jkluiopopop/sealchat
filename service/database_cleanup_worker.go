package service

import (
	"log"
	"sync"
	"time"
)

var databaseCleanupWorkerOnce sync.Once

func StartDatabaseCleanupWorker() {
	databaseCleanupWorkerOnce.Do(func() {
		log.Println("db-cleanup: worker 启动")
		go runDatabaseCleanupWorker()
	})
}

func runDatabaseCleanupWorker() {
	runDatabaseCleanup(time.Now())

	ticker := time.NewTicker(DatabaseCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		runDatabaseCleanup(time.Now())
	}
}

func runDatabaseCleanup(now time.Time) {
	report, err := RunDefaultDatabaseCleanup(now)
	if err != nil {
		log.Printf("db-cleanup: 执行失败: %v", err)
		return
	}
	if report == nil {
		return
	}
	for _, item := range report.Results {
		if item.AffectedRows <= 0 {
			continue
		}
		log.Printf("db-cleanup: %s 清理 %d 条", item.Name, item.AffectedRows)
	}
}
