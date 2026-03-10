package bot

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"fleet-management/internal/db"
	"fleet-management/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotState struct {
	Step      string
	Repair    models.Repair
	CarNumber string
}

var userState = make(map[int64]*BotState)
var TgBot *tgbotapi.BotAPI

const (
	StepIdle          = "idling"
	StepChoosingCar   = "choosing_car"
	StepVenterMileage = "enter_mileage"
	StepEnterDesc     = "enter_desc"
	StepEnterPrice    = "enter_price"
	StepEnterExecutor = "enter_executor"
	StepEnterNotes    = "enter_notes"
	StepSearchCar     = "search_car"
	// Tire change steps
	StepTireMileage  = "tire_mileage"
	StepTireDesc     = "tire_desc"
	StepTirePrice    = "tire_price"
	StepTireExecutor = "tire_executor"
	StepTireNotes    = "tire_notes"

	StepRepairVAT = "enter_repair_vat"
	StepTireVAT   = "enter_tire_vat"

	StepConfirmRepair = "confirm_repair"
	StepConfirmTire   = "confirm_tire"
)

func StartBot(token string) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	go func() {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		updates := bot.GetUpdatesChan(u)

		for update := range updates {
			if update.Message != nil {
				handleMessage(bot, update.Message)
			} else if update.CallbackQuery != nil {
				handleCallback(bot, update.CallbackQuery)
			}
		}
	}()

	return bot
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	state, exists := userState[chatID]

	if !exists || msg.Text == "/start" {
		state = &BotState{Step: StepIdle}
		userState[chatID] = state
	}

	if !isAuthorized(chatID) {
		bot.Send(tgbotapi.NewMessage(chatID, "Ви не авторизовані для використання цього бота."))
		return
	}

	// Global menu buttons - allow switching flow anytime
	if strings.HasPrefix(msg.Text, "/add_repair") || msg.Text == "🔧 Додати ремонт" {
		startRepairFlow(bot, chatID)
		return
	} else if msg.Text == "🛞 Додати резину" {
		startTireFlow(bot, chatID)
		return
	} else if msg.Text == "🔍 Пошук авто" {
		showCarSelection(bot, chatID, "Виберіть автомобіль:", "choosing_car_search")
		return
	} else if msg.Text == "📦 Документи" {
		sendDocumentsStatus(bot, chatID)
		return
	} else if msg.Text == "❌ Скасувати" {
		state.Step = StepIdle
		msg := tgbotapi.NewMessage(chatID, "Дію скасовано.")
		msg.ReplyMarkup = getMainKeyboard()
		bot.Send(msg)
		return
	}

	switch state.Step {
	case StepIdle:
		// Check if user entered a short plate number directly for repair flow
		if len(msg.Text) >= 4 {
			car, err := db.FindCarSmart(msg.Text)
			if err == nil {
				state.Step = StepVenterMileage
				state.Repair = models.Repair{
					CarID:     car.ID,
					CarNumber: car.Number,
					Date:      time.Now(),
				}
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Вибрано машину: %s. Введіть пробіг (км):", car.Number)))
				return
			}
		}
		msg := tgbotapi.NewMessage(chatID, "Привіт! Скористайтесь меню.")
		msg.ReplyMarkup = getMainKeyboard()
		bot.Send(msg)

	case StepSearchCar:
		car, err := db.FindCarSmart(msg.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Авто не знайдено. Спробуйте ще раз або виберіть іншу опцію з меню."))
			state.Step = StepIdle
			return
		}
		sendCarCard(bot, chatID, car)
		state.Step = StepIdle

	case StepTireMileage:
		mileage, err := strconv.Atoi(msg.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, введіть число:"))
			return
		}
		// We can reuse the Repair field in BotState for tire change logic or add a new field.
		// For simplicity, let's just use it to store temporary data or add a Tire field to BotState.
		// Let's assume we use BotState's Repair for now as they are similar (Mileage, Price, Desc).
		state.Repair.Mileage = mileage
		state.Step = StepTireDesc
		msg := tgbotapi.NewMessage(chatID, "Введіть опис (наприклад, 'Зимова резина'):")
		msg.ReplyMarkup = getCancelKeyboard()
		bot.Send(msg)

	case StepTireDesc:
		state.Repair.Description = msg.Text
		state.Step = StepTirePrice
		msg := tgbotapi.NewMessage(chatID, "Введіть ціну (грн):")
		msg.ReplyMarkup = getCancelKeyboard()
		bot.Send(msg)

	case StepTirePrice:
		price, err := db.EvaluatePrice(msg.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, введіть число (можна вираз, напр. 2500+1000):"))
			return
		}
		state.Repair.Price = price
		state.Step = StepTireVAT
		msg := tgbotapi.NewMessage(chatID, "Ціна з ПДВ?")
		msg.ReplyMarkup = getYesNoKeyboard()
		bot.Send(msg)

	case StepTireExecutor:
		if msg.Text != "⏩ Пропустити" {
			state.Repair.Executor = msg.Text
		}
		state.Step = StepTireNotes
		msg := tgbotapi.NewMessage(chatID, "Введіть примітку (або пропустіть):")
		msg.ReplyMarkup = getSkipKeyboard()
		bot.Send(msg)

	case StepTireNotes:
		if msg.Text != "⏩ Пропустити" {
			state.Repair.Notes = msg.Text
		}
		state.Step = StepConfirmTire

		summary := fmt.Sprintf("📝 *Підтвердіть заміну резини:*\n\n"+
			"🚗 Авто: %s\n"+
			"🛣 Пробіг: %d км\n"+
			"📄 Опис: %s\n"+
			"🔧 Виконавець: %s\n"+
			"💰 Ціна: %.2f грн %s\n"+
			"📝 Примітки: %s",
			state.Repair.CarNumber, state.Repair.Mileage, state.Repair.Description,
			getVal(state.Repair.Executor), state.Repair.Price, getVATStr(state.Repair.IsVAT), getVal(state.Repair.Notes))

		msg := tgbotapi.NewMessage(chatID, summary)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = getConfirmationKeyboard()
		bot.Send(msg)

	case StepVenterMileage:
		mileage, err := strconv.Atoi(msg.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, введіть число:"))
			return
		}
		state.Repair.Mileage = mileage
		state.Step = StepEnterDesc
		msg := tgbotapi.NewMessage(chatID, "Введіть опис ремонту:")
		msg.ReplyMarkup = getCancelKeyboard()
		bot.Send(msg)

	case StepEnterDesc:
		state.Repair.Description = msg.Text
		state.Step = StepEnterPrice
		msg := tgbotapi.NewMessage(chatID, "Введіть ціну (грн):")
		msg.ReplyMarkup = getCancelKeyboard()
		bot.Send(msg)

	case StepEnterPrice:
		price, err := db.EvaluatePrice(msg.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Будь ласка, введіть число (можна вираз, напр. 2500+1000):"))
			return
		}
		state.Repair.Price = price
		state.Step = StepRepairVAT
		msg := tgbotapi.NewMessage(chatID, "Ціна з ПДВ?")
		msg.ReplyMarkup = getYesNoKeyboard()
		bot.Send(msg)

	case StepEnterExecutor:
		if msg.Text != "⏩ Пропустити" {
			state.Repair.Executor = msg.Text
		}
		state.Step = StepEnterNotes
		msg := tgbotapi.NewMessage(chatID, "Введіть примітку (або пропустіть):")
		msg.ReplyMarkup = getSkipKeyboard()
		bot.Send(msg)

	case StepEnterNotes:
		if msg.Text != "⏩ Пропустити" {
			state.Repair.Notes = msg.Text
		}
		state.Step = StepConfirmRepair

		summary := fmt.Sprintf("📝 *Підтвердіть ремонт:*\n\n"+
			"🚗 Авто: %s\n"+
			"🛣 Пробіг: %d км\n"+
			"📄 Опис: %s\n"+
			"🔧 Виконавець: %s\n"+
			"💰 Ціна: %.2f грн %s\n"+
			"📝 Примітки: %s",
			state.Repair.CarNumber, state.Repair.Mileage, state.Repair.Description,
			getVal(state.Repair.Executor), state.Repair.Price, getVATStr(state.Repair.IsVAT), getVal(state.Repair.Notes))

		msg := tgbotapi.NewMessage(chatID, summary)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = getConfirmationKeyboard()
		bot.Send(msg)

	case StepRepairVAT:
		state.Repair.IsVAT = (msg.Text == "Так ✅")
		state.Step = StepEnterExecutor
		msg := tgbotapi.NewMessage(chatID, "Введіть виконавця (наприклад, 'СТО Сінтайл'):")
		msg.ReplyMarkup = getSkipKeyboard()
		bot.Send(msg)

	case StepTireVAT:
		state.Repair.IsVAT = (msg.Text == "Так ✅")
		state.Step = StepTireExecutor
		msg := tgbotapi.NewMessage(chatID, "Введіть виконавця (наприклад, 'СТО Сінтайл'):")
		msg.ReplyMarkup = getSkipKeyboard()
		bot.Send(msg)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	chatID := cb.From.ID
	state := userState[chatID]
	if state == nil {
		state = &BotState{Step: StepIdle}
		userState[chatID] = state
	}

	if strings.HasPrefix(cb.Data, "car_") {
		num := strings.TrimPrefix(cb.Data, "car_")
		car, err := db.FindCarSmart(num)
		if err != nil {
			bot.Request(tgbotapi.NewCallback(cb.ID, "Машину не знайдено"))
			return
		}

		if state.Step == StepChoosingCar {
			state.Step = StepVenterMileage
			state.Repair = models.Repair{CarID: car.ID, CarNumber: car.Number, Date: time.Now()}
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔧 Ремонт для: %s. Введіть пробіг:", car.Number))
			msg.ReplyMarkup = getCancelKeyboard()
			bot.Send(msg)
		} else if state.Step == "choosing_car_tire" {
			state.Step = StepTireMileage
			state.Repair = models.Repair{CarID: car.ID, CarNumber: car.Number, Date: time.Now()}
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🛞 Заміна резини для: %s. Введіть пробіг:", car.Number))
			msg.ReplyMarkup = getCancelKeyboard()
			bot.Send(msg)
		} else {
			sendCarCard(bot, chatID, car)
		}
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	} else if strings.HasPrefix(cb.Data, "add_rep_") {
		num := strings.TrimPrefix(cb.Data, "add_rep_")
		car, _ := db.FindCarSmart(num)
		state.Step = StepVenterMileage
		state.Repair = models.Repair{CarID: car.ID, CarNumber: car.Number, Date: time.Now()}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔧 Ремонт для: %s. Введіть пробіг:", car.Number))
		msg.ReplyMarkup = getCancelKeyboard()
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	} else if strings.HasPrefix(cb.Data, "add_tire_") {
		num := strings.TrimPrefix(cb.Data, "add_tire_")
		car, _ := db.FindCarSmart(num)
		state.Step = StepTireMileage
		state.Repair = models.Repair{CarID: car.ID, CarNumber: car.Number, Date: time.Now()}
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🛞 Заміна резини для: %s. Введіть пробіг:", car.Number))
		msg.ReplyMarkup = getCancelKeyboard()
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	} else if cb.Data == "confirm_yes" {
		if state.Step == StepConfirmRepair {
			if err := db.DB.Create(&state.Repair).Error; err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Помилка при збереженні: "+err.Error()))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Ремонт для %s успішно додано!", state.Repair.CarNumber)))
			}
		} else if state.Step == StepConfirmTire {
			tire := models.TireChange{
				CarID:       state.Repair.CarID,
				CarNumber:   state.Repair.CarNumber,
				Date:        time.Now(),
				Mileage:     state.Repair.Mileage,
				Description: state.Repair.Description,
				Executor:    state.Repair.Executor,
				Price:       state.Repair.Price,
				IsVAT:       state.Repair.IsVAT,
				Notes:       state.Repair.Notes,
			}
			if err := db.DB.Create(&tire).Error; err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Помилка при збереженні: "+err.Error()))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Заміну резини для %s успішно додано!", tire.CarNumber)))
			}
		}
		state.Step = StepIdle
		bot.Request(tgbotapi.NewCallback(cb.ID, "Збережено"))
	} else if cb.Data == "confirm_no" {
		state.Step = StepIdle
		msg := tgbotapi.NewMessage(chatID, "❌ Дію скасовано.")
		msg.ReplyMarkup = getMainKeyboard()
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(cb.ID, "Скасовано"))
	}
}

func startRepairFlow(bot *tgbotapi.BotAPI, chatID int64) {
	showCarSelection(bot, chatID, "Виберіть автомобіль для ремонту:", StepChoosingCar)
}

func startTireFlow(bot *tgbotapi.BotAPI, chatID int64) {
	showCarSelection(bot, chatID, "Виберіть автомобіль для заміни резини:", "choosing_car_tire")
}

func showCarSelection(bot *tgbotapi.BotAPI, chatID int64, text string, step string) {
	var cars []models.Car
	db.DB.Find(&cars)
	log.Printf("Found %d cars for selection", len(cars))

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, car := range cars {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s (%s)", car.Number, car.Model), "car_"+car.Number),
		)
		rows = append(rows, row)
	}
	log.Printf("Created %d rows for keyboard", len(rows))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	bot.Send(msg)

	state, exists := userState[chatID]
	if !exists {
		state = &BotState{}
		userState[chatID] = state
	}
	state.Step = step
}

func sendDocumentsStatus(bot *tgbotapi.BotAPI, chatID int64) {
	var docs []models.Document
	db.DB.Order("expiry_date asc").Find(&docs)

	if len(docs) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Документів не знайдено."))
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Статус документів:* \n\n")

	now := time.Now()
	for _, doc := range docs {
		daysLeft := int(doc.ExpiryDate.Sub(now).Hours() / 24)
		statusIcon := "✅"
		if daysLeft < 0 {
			statusIcon = "❌"
		} else if daysLeft < 30 {
			statusIcon = "⚠️"
		}

		daysStr := ""
		if daysLeft < 0 {
			daysStr = fmt.Sprintf("(прострочено на %d дн.)", -daysLeft)
		} else {
			daysStr = fmt.Sprintf("(залишилось %d дн.)", daysLeft)
		}

		sb.WriteString(fmt.Sprintf("%s *%s*\n%s: до %s %s\n\n",
			statusIcon, doc.CarNumber, doc.Type, doc.ExpiryDate.Format("02.01.2006"), daysStr))
	}

	msg := tgbotapi.NewMessage(chatID, sb.String())
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func sendCarCard(bot *tgbotapi.BotAPI, chatID int64, car *models.Car) {
	var docs []models.Document
	db.DB.Where("car_id = ?", car.ID).Find(&docs)

	docStatus := "✅ Документи в порядку"
	for _, doc := range docs {
		if doc.ExpiryDate.Before(time.Now()) {
			docStatus = "❌ Є прострочені документи!"
			break
		}
	}

	// Fetch history
	var repairs []models.Repair
	db.DB.Where("car_id = ?", car.ID).Order("date DESC").Limit(3).Find(&repairs)

	var tires []models.TireChange
	db.DB.Where("car_id = ?", car.ID).Order("date DESC").Limit(3).Find(&tires)

	text := fmt.Sprintf("🚗 *Картка авто: %s*\n\n"+
		"📍 Модель: %s\n"+
		"📅 Рік: %d\n"+
		"👤 Власник: %s\n\n"+
		"📄 Статус документів: %s\n\n",
		car.Number, car.Model, car.Year, car.Owner, docStatus)

	if len(repairs) > 0 {
		text += "🔧 *Останні ремонти:*\n"
		for _, r := range repairs {
			text += fmt.Sprintf("• %s: %s (%.0f ₴)\n", r.Date.Format("02.01.2006"), r.Description, r.Price)
		}
		text += "\n"
	}

	if len(tires) > 0 {
		text += "🛞 *Остання резина:*\n"
		for _, t := range tires {
			text += fmt.Sprintf("• %s: %s (%.0f ₴)\n", t.Date.Format("02.01.2006"), t.Description, t.Price)
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	// Inline buttons for quick actions
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔧 +Ремонт", "add_rep_"+car.Number),
		tgbotapi.NewInlineKeyboardButtonData("🛞 +Резина", "add_tire_"+car.Number),
	)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)

	bot.Send(msg)
}

func isAuthorized(chatID int64) bool {
	ids := os.Getenv("AUTHORIZED_USER_IDS")
	if ids == "" {
		return true // Default to allow if not configured, for safety we might want to change this later
	}
	for _, idStr := range strings.Split(ids, ",") {
		id, _ := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if id == chatID {
			return true
		}
	}
	return false
}

func getMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔍 Пошук авто"),
			tgbotapi.NewKeyboardButton("📦 Документи"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔧 Додати ремонт"),
			tgbotapi.NewKeyboardButton("🛞 Додати резину"),
		),
	)
}

func getConfirmationKeyboard() tgbotapi.InlineKeyboardMarkup {
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Підтвердити", "confirm_yes"),
		tgbotapi.NewInlineKeyboardButtonData("❌ Скасувати", "confirm_no"),
	)
	return tgbotapi.NewInlineKeyboardMarkup(row)
}

func getCancelKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Скасувати"),
		),
	)
}

func getSkipKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⏩ Пропустити"),
			tgbotapi.NewKeyboardButton("❌ Скасувати"),
		),
	)
}

func getYesNoKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Так ✅"),
			tgbotapi.NewKeyboardButton("Ні ❌"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Скасувати"),
		),
	)
}

func getVATStr(isVAT bool) string {
	if isVAT {
		return "(з ПДВ)"
	}
	return "(без ПДВ)"
}

func getVal(s string) string {
	if s == "" {
		return " — "
	}
	return s
}
