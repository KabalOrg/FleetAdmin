package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"fleet-management/internal/db"
	"fleet-management/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartReminderWorker runs a background task that checks for expiring documents at the specified time daily.
func StartReminderWorker(bot *tgbotapi.BotAPI) {
	scheduleNextReminder(bot)
}

func scheduleNextReminder(bot *tgbotapi.BotAPI) {
	reminderTimeStr := db.GetSetting("reminder_time", "09:00")
	now := time.Now()

	// Parse reminder time
	parts := strings.Split(reminderTimeStr, ":")
	if len(parts) != 2 {
		log.Println("Invalid reminder_time format, using 09:00")
		reminderTimeStr = "09:00"
		parts = []string{"09", "00"}
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

	// Calculate next reminder time
	nextReminder := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if nextReminder.Before(now) {
		// If time has passed today, schedule for tomorrow
		nextReminder = nextReminder.AddDate(0, 0, 1)
	}

	duration := nextReminder.Sub(now)
	log.Printf("Next reminder scheduled at %s (in %v)", nextReminder.Format("2006-01-02 15:04"), duration)

	time.AfterFunc(duration, func() {
		checkAndSendReminders(bot)
		scheduleNextReminder(bot) // Schedule the next one
	})
}

func checkAndSendReminders(bot *tgbotapi.BotAPI) {
	log.Println("Checking for expiring documents...")

	reminderDaysStr := db.GetSetting("reminder_days", "30")
	reminderDays, _ := strconv.Atoi(reminderDaysStr)
	if reminderDays == 0 {
		reminderDays = 30
	}

	adminIDsStr := db.GetSetting("admin_chat_ids", "")
	if adminIDsStr == "" {
		log.Println("No admin chat IDs configured for reminders")
		return
	}

	var docs []models.Document
	now := time.Now()
	threshold := now.AddDate(0, 0, reminderDays)

	// Find documents expiring within threshold
	db.DB.Where("expiry_date <= ?", threshold).Find(&docs)

	if len(docs) == 0 {
		return
	}

	// Group by car for better formatting
	expiringByCar := make(map[string][]models.Document)
	for _, doc := range docs {
		expiringByCar[doc.CarNumber] = append(expiringByCar[doc.CarNumber], doc)
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString("🔔 *Нагадування про закінчення терміну документів:*\n\n")

	for carNum, carDocs := range expiringByCar {
		messageBuilder.WriteString(fmt.Sprintf("🚗 *Авто: %s*\n", carNum))
		for _, doc := range carDocs {
			daysLeft := int(doc.ExpiryDate.Sub(now).Hours() / 24)
			statusEmoji := "⚠️"
			if daysLeft < 0 {
				statusEmoji = "❌"
			}
			messageBuilder.WriteString(fmt.Sprintf("%s %s: до %s (%d дн.)\n",
				statusEmoji, doc.Type, doc.ExpiryDate.Format("02.01.2006"), daysLeft))
		}
		messageBuilder.WriteString("\n")
	}

	text := messageBuilder.String()
	for _, idStr := range strings.Split(adminIDsStr, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err != nil {
			continue
		}
		msg := tgbotapi.NewMessage(id, text)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	}
}
