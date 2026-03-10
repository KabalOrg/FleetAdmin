package backup

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartBackupWorker(bot *tgbotapi.BotAPI, dbPath string, chatID int64) {
	now := time.Now()
	nextBackup := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
	if now.After(nextBackup) {
		nextBackup = nextBackup.Add(24 * time.Hour)
	}
	duration := nextBackup.Sub(now)

	log.Printf("Next backup scheduled at %s", nextBackup.Format("2006-01-02 15:04:05"))

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	time.AfterFunc(duration, func() {
		err := PerformBackupAndSend(bot, dbPath, chatID)
		if err != nil {
			log.Printf("Backup failed: %v", err)
		}
		for range ticker.C {
			err := PerformBackupAndSend(bot, dbPath, chatID)
			if err != nil {
				log.Printf("Backup failed: %v", err)
			}
		}
	})
}

func PerformBackupAndSend(bot *tgbotapi.BotAPI, dbPath string, chatID int64) error {
	timestamp := time.Now().Format("2006-01-02_15-04")
	backupPath := fmt.Sprintf("backup_%s.db", timestamp)

	// Скопировать файл БД
	err := copyFile(dbPath, backupPath)
	if err != nil {
		return fmt.Errorf("failed to copy database: %w", err)
	}

	// Вычислить MD5 хэш бэкапа
	hash, err := computeMD5(backupPath)
	if err != nil {
		log.Printf("Warning: failed to compute MD5 for backup: %v", err)
	} else {
		log.Printf("Backup file MD5: %s", hash)
	}

	// Отправить в Telegram
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(backupPath))
	doc.Caption = fmt.Sprintf("Бэкап базы данных от %s", timestamp)

	_, err = bot.Send(doc)
	if err != nil {
		return fmt.Errorf("failed to send backup to Telegram: %w", err)
	}

	// Удалить локальный файл бэкапа после отправки
	os.Remove(backupPath)

	log.Printf("Backup sent to Telegram successfully")
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func computeMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
