package handlers

import (
	"crypto/md5"
	"fleet-management/internal/backup"
	"fleet-management/internal/bot"
	"fleet-management/internal/db"
	"fleet-management/internal/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetRepairs(c *gin.Context) {
	var repairs []models.Repair
	db.DB.Order("date desc").Find(&repairs)

	var totalCount int64
	db.DB.Model(&models.Repair{}).Count(&totalCount)

	c.HTML(http.StatusOK, "repairs.html", gin.H{
		"title":      "Ремонти",
		"repairs":    repairs,
		"totalCount": totalCount,
	})
}

func GetFleet(c *gin.Context) {
	var cars []models.Car
	db.DB.Find(&cars)
	c.HTML(http.StatusOK, "fleet.html", gin.H{
		"title": "Автопарк",
		"cars":  cars,
	})
}

func GetDocuments(c *gin.Context) {
	var docs []models.Document
	db.DB.Find(&docs)

	// Enrich with dynamic "Days Left" logic
	type DocView struct {
		models.Document
		DaysLeft int
		Status   string
		Class    string
	}
	var docViews []DocView
	now := time.Now()

	for _, d := range docs {
		days := int(d.ExpiryDate.Sub(now).Hours() / 24)
		class := ""
		status := "Активний"

		if days < 0 {
			class = "bg-rose-500/20 text-rose-400"
			status = "Прострочений"
		} else if days < 30 {
			class = "bg-yellow-500/20 text-yellow-500"
		} else {
			class = "bg-emerald-500/20 text-emerald-400"
		}

		docViews = append(docViews, DocView{
			Document: d,
			DaysLeft: days,
			Status:   status,
			Class:    class,
		})
	}

	c.HTML(http.StatusOK, "documents.html", gin.H{
		"title": "Документи",
		"docs":  docViews,
	})
}
func DeleteCar(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Car{}, id)
	c.Redirect(303, "/fleet")
}

// GetCarDetail fetches detailed info (car, repairs, tires) for a single car
func GetCarDetail(c *gin.Context) {
	id := c.Param("id")
	var car models.Car
	if err := db.DB.First(&car, id).Error; err != nil {
		c.String(http.StatusNotFound, "Авто не знайдено")
		return
	}

	var repairs []models.Repair
	db.DB.Where("car_number = ? OR car_id = ?", car.Number, id).Order("date DESC").Find(&repairs)

	var tires []models.TireChange
	db.DB.Where("car_number = ? OR car_id = ?", car.Number, id).Order("date DESC").Find(&tires)

	c.HTML(http.StatusOK, "car_detail.html", gin.H{
		"title":   "Картка авто: " + car.Number,
		"car":     car,
		"repairs": repairs,
		"tires":   tires,
	})
}

// SearchCars returns a JSON array of cars matching the query
func SearchCars(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusOK, []models.Car{})
		return
	}

	var cars []models.Car
	// Search by number or model
	searchTerm := "%" + query + "%"
	db.DB.Where("number LIKE ? OR model LIKE ?", searchTerm, searchTerm).Limit(10).Find(&cars)

	c.JSON(http.StatusOK, cars)
}

func DeleteDocument(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Document{}, id)
	c.Redirect(303, "/documents")
}

func DeleteRepair(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Deleting repair with ID: %s", id)
	if err := db.DB.Delete(&models.Repair{}, id).Error; err != nil {
		log.Printf("Error deleting repair: %v", err)
		c.String(500, "Error deleting repair")
		return
	}

	redirectTo := c.Query("redirect")
	if redirectTo == "duplicates" {
		c.Redirect(303, "/repairs/duplicates")
	} else {
		c.Redirect(303, "/repairs")
	}
}

func DeleteTire(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Deleting tire change with ID: %s", id)
	if err := db.DB.Delete(&models.TireChange{}, id).Error; err != nil {
		log.Printf("Error deleting tire: %v", err)
		c.String(500, "Error deleting tire")
		return
	}

	redirectTo := c.Query("redirect")
	if redirectTo == "duplicates" {
		c.Redirect(303, "/repairs/duplicates")
	} else {
		c.Redirect(303, "/tires")
	}
}

func GetTires(c *gin.Context) {
	var tires []models.TireChange
	db.DB.Order("date desc").Find(&tires)

	var totalCount int64
	db.DB.Model(&models.TireChange{}).Count(&totalCount)

	c.HTML(http.StatusOK, "tires.html", gin.H{
		"title":      "Заміна Резини",
		"tires":      tires,
		"totalCount": totalCount,
	})
}
func AddCar(c *gin.Context) {
	c.HTML(http.StatusOK, "car_form.html", gin.H{
		"title": "Додати Авто",
	})
}

func CreateCar(c *gin.Context) {
	car := models.Car{
		Model:  c.PostForm("model"),
		Number: strings.ToUpper(c.PostForm("number")),
		Owner:  c.PostForm("owner"),
	}
	parsedYear, _ := strconv.Atoi(c.PostForm("year"))
	car.Year = parsedYear

	db.DB.Create(&car)
	c.Redirect(303, "/fleet")
}

func EditCar(c *gin.Context) {
	id := c.Param("id")
	var car models.Car
	if err := db.DB.First(&car, id).Error; err != nil {
		c.Redirect(303, "/fleet")
		return
	}
	c.HTML(http.StatusOK, "car_form.html", gin.H{
		"title": "Редагувати Авто",
		"car":   car,
	})
}

func UpdateCar(c *gin.Context) {
	id := c.Param("id")
	var car models.Car
	if err := db.DB.First(&car, id).Error; err != nil {
		c.Redirect(303, "/fleet")
		return
	}

	car.Model = c.PostForm("model")
	car.Number = strings.ToUpper(c.PostForm("number"))
	parsedYear, _ := strconv.Atoi(c.PostForm("year"))
	car.Year = parsedYear
	car.Owner = c.PostForm("owner")

	db.DB.Save(&car)
	c.Redirect(303, "/fleet")
}

func AddDocument(c *gin.Context) {
	var cars []models.Car
	db.DB.Find(&cars)
	var docTypes []models.DocumentType
	db.DB.Find(&docTypes)
	c.HTML(http.StatusOK, "document_form.html", gin.H{
		"title":    "Додати документ",
		"cars":     cars,
		"docTypes": docTypes,
	})
}

func CreateDocument(c *gin.Context) {
	issueDate, _ := time.Parse("2006-01-02", c.PostForm("issue_date"))
	expiryDate, _ := time.Parse("2006-01-02", c.PostForm("expiry_date"))

	doc := models.Document{
		CarNumber:  c.PostForm("car_number"),
		Type:       c.PostForm("type"),
		IssueDate:  issueDate,
		ExpiryDate: expiryDate,
		Status:     "Active",
	}

	car, err := db.FindCarSmart(doc.CarNumber)
	if err == nil {
		doc.CarID = car.ID
		doc.CarNumber = car.Number
	}

	db.DB.Create(&doc)
	c.Redirect(303, "/documents")
}

func EditDocument(c *gin.Context) {
	id := c.Param("id")
	var doc models.Document
	if err := db.DB.First(&doc, id).Error; err != nil {
		c.Redirect(303, "/documents")
		return
	}
	var cars []models.Car
	db.DB.Find(&cars)
	var docTypes []models.DocumentType
	db.DB.Find(&docTypes)
	c.HTML(http.StatusOK, "document_form.html", gin.H{
		"title":    "Редагувати документ",
		"doc":      doc,
		"cars":     cars,
		"docTypes": docTypes,
	})
}

func UpdateDocument(c *gin.Context) {
	id := c.Param("id")
	var doc models.Document
	if err := db.DB.First(&doc, id).Error; err != nil {
		c.Redirect(303, "/documents")
		return
	}

	issueDate, _ := time.Parse("2006-01-02", c.PostForm("issue_date"))
	expiryDate, _ := time.Parse("2006-01-02", c.PostForm("expiry_date"))

	doc.CarNumber = c.PostForm("car_number")
	doc.Type = c.PostForm("type")
	doc.IssueDate = issueDate
	doc.ExpiryDate = expiryDate

	car, err := db.FindCarSmart(doc.CarNumber)
	if err == nil {
		doc.CarID = car.ID
		doc.CarNumber = car.Number
	}

	db.DB.Save(&doc)
	c.Redirect(303, "/documents")
}

func AddRepair(c *gin.Context) {
	var cars []models.Car
	db.DB.Find(&cars)

	var executors []models.Executor
	db.DB.Find(&executors)

	c.HTML(http.StatusOK, "repair_form.html", gin.H{
		"title":     "Додати ремонт",
		"cars":      cars,
		"executors": executors,
	})
}

func CreateRepair(c *gin.Context) {
	date, _ := time.Parse("2006-01-02", c.PostForm("date"))
	mileage, _ := strconv.Atoi(c.PostForm("mileage"))
	price, _ := db.EvaluatePrice(c.PostForm("price"))
	isVAT := c.PostForm("is_vat") == "on"

	repair := models.Repair{
		Date:        date,
		CarNumber:   c.PostForm("car_number"),
		Mileage:     mileage,
		Description: c.PostForm("description"),
		Executor:    c.PostForm("executor"),
		Price:       price,
		IsVAT:       isVAT,
		Notes:       c.PostForm("notes"),
	}

	// Try to find car ID for linking
	car, err := db.FindCarSmart(repair.CarNumber)
	if err == nil {
		repair.CarID = car.ID
		repair.CarNumber = car.Number
	}

	db.DB.Create(&repair)
	c.Redirect(303, "/repairs")
}

func AddTire(c *gin.Context) {
	var cars []models.Car
	db.DB.Find(&cars)

	var executors []models.Executor
	db.DB.Find(&executors)

	c.HTML(http.StatusOK, "tire_form.html", gin.H{
		"title":     "Додати заміну резини",
		"cars":      cars,
		"executors": executors,
	})
}

func CreateTire(c *gin.Context) {
	date, _ := time.Parse("2006-01-02", c.PostForm("date"))
	mileage, _ := strconv.Atoi(c.PostForm("mileage"))
	price, _ := db.EvaluatePrice(c.PostForm("price"))
	isVAT := c.PostForm("is_vat") == "on"

	tire := models.TireChange{
		Date:        date,
		CarNumber:   c.PostForm("car_number"),
		Mileage:     mileage,
		Description: c.PostForm("description"),
		Executor:    c.PostForm("executor"),
		Price:       price,
		IsVAT:       isVAT,
		Notes:       c.PostForm("notes"),
	}

	car, err := db.FindCarSmart(tire.CarNumber)
	if err == nil {
		tire.CarID = car.ID
		tire.CarNumber = car.Number
	}

	db.DB.Create(&tire)
	c.Redirect(303, "/tires")
}

func EditRepair(c *gin.Context) {
	id := c.Param("id")
	var repair models.Repair
	if err := db.DB.First(&repair, id).Error; err != nil {
		c.Redirect(303, "/repairs")
		return
	}
	var cars []models.Car
	db.DB.Find(&cars)
	var executors []models.Executor
	db.DB.Find(&executors)
	c.HTML(http.StatusOK, "repair_form.html", gin.H{
		"title":     "Редагувати ремонт",
		"repair":    repair,
		"cars":      cars,
		"executors": executors,
	})
}

func UpdateRepair(c *gin.Context) {
	id := c.Param("id")
	var repair models.Repair
	if err := db.DB.First(&repair, id).Error; err != nil {
		c.Redirect(303, "/repairs")
		return
	}

	date, _ := time.Parse("2006-01-02", c.PostForm("date"))
	mileage, _ := strconv.Atoi(c.PostForm("mileage"))
	price, _ := db.EvaluatePrice(c.PostForm("price"))
	isVAT := c.PostForm("is_vat") == "on"

	repair.Date = date
	repair.CarNumber = c.PostForm("car_number")
	repair.Mileage = mileage
	repair.Description = c.PostForm("description")
	repair.Executor = c.PostForm("executor")
	repair.Price = price
	repair.IsVAT = isVAT
	repair.Notes = c.PostForm("notes")

	car, err := db.FindCarSmart(repair.CarNumber)
	if err == nil {
		repair.CarID = car.ID
		repair.CarNumber = car.Number
	}

	db.DB.Save(&repair)
	c.Redirect(303, "/repairs")
}

func EditTire(c *gin.Context) {
	id := c.Param("id")
	var tire models.TireChange
	if err := db.DB.First(&tire, id).Error; err != nil {
		c.Redirect(303, "/tires")
		return
	}
	var cars []models.Car
	db.DB.Find(&cars)

	var executors []models.Executor
	db.DB.Find(&executors)

	c.HTML(http.StatusOK, "tire_form.html", gin.H{
		"title":     "Редагувати заміну резини",
		"tire":      tire,
		"cars":      cars,
		"executors": executors,
	})
}

func UpdateTire(c *gin.Context) {
	id := c.Param("id")
	var tire models.TireChange
	if err := db.DB.First(&tire, id).Error; err != nil {
		c.Redirect(303, "/tires")
		return
	}

	date, _ := time.Parse("2006-01-02", c.PostForm("date"))
	mileage, _ := strconv.Atoi(c.PostForm("mileage"))
	price, _ := db.EvaluatePrice(c.PostForm("price"))
	isVAT := c.PostForm("is_vat") == "on"

	tire.Date = date
	tire.CarNumber = c.PostForm("car_number")
	tire.Mileage = mileage
	tire.Description = c.PostForm("description")
	tire.Executor = c.PostForm("executor")
	tire.Price = price
	tire.IsVAT = isVAT
	tire.Notes = c.PostForm("notes")

	car, err := db.FindCarSmart(tire.CarNumber)
	if err == nil {
		tire.CarID = car.ID
		tire.CarNumber = car.Number
	}

	db.DB.Save(&tire)
	c.Redirect(303, "/tires")
}

// GetExecutors returns a list of all executors
func GetExecutors(c *gin.Context) {
	var executors []models.Executor
	db.DB.Find(&executors)
	c.HTML(http.StatusOK, "executors.html", gin.H{
		"title":     "Виконавці",
		"executors": executors,
	})
}

// AddExecutor renders the new executor form
func AddExecutor(c *gin.Context) {
	c.HTML(http.StatusOK, "executor_form.html", gin.H{
		"title": "Додати виконавця",
	})
}

// CreateExecutor handles the creation of a new executor
func CreateExecutor(c *gin.Context) {
	executor := models.Executor{
		Name:  c.PostForm("name"),
		Phone: c.PostForm("phone"),
		Notes: c.PostForm("notes"),
	}

	db.DB.Create(&executor)
	c.Redirect(303, "/executors")
}

// EditExecutor renders the edit executor form
func EditExecutor(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var executor models.Executor
	if err := db.DB.First(&executor, id).Error; err != nil {
		c.String(http.StatusNotFound, "Виконавця не знайдено")
		return
	}

	c.HTML(http.StatusOK, "executor_form.html", gin.H{
		"title":    "Редагувати виконавця",
		"executor": executor,
	})
}

// UpdateExecutor handles updating an existing executor
func UpdateExecutor(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var executor models.Executor
	if err := db.DB.First(&executor, id).Error; err != nil {
		c.String(http.StatusNotFound, "Виконавця не знайдено")
		return
	}

	executor.Name = c.PostForm("name")
	executor.Phone = c.PostForm("phone")
	executor.Notes = c.PostForm("notes")

	db.DB.Save(&executor)
	c.Redirect(303, "/executors")
}

// DeleteExecutor handles deleting an executor
func DeleteExecutor(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	db.DB.Delete(&models.Executor{}, id)
	c.Redirect(303, "/executors")
}

// ─── Document Types CRUD ───────────────────────────────────────────────────

// GetDocumentTypes lists all document types
func GetDocumentTypes(c *gin.Context) {
	var types []models.DocumentType
	db.DB.Find(&types)
	c.HTML(http.StatusOK, "document_types.html", gin.H{
		"title": "Типи документів",
		"types": types,
	})
}

// AddDocumentType renders the creation form
func AddDocumentType(c *gin.Context) {
	c.HTML(http.StatusOK, "document_type_form.html", gin.H{
		"title": "Додати тип документа",
	})
}

// CreateDocumentType handles the form POST
func CreateDocumentType(c *gin.Context) {
	dt := models.DocumentType{
		Name: strings.TrimSpace(c.PostForm("name")),
	}
	db.DB.Create(&dt)
	c.Redirect(303, "/document-types")
}

// EditDocumentType renders the edit form
func EditDocumentType(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var dt models.DocumentType
	if err := db.DB.First(&dt, id).Error; err != nil {
		c.String(http.StatusNotFound, "Тип не знайдено")
		return
	}
	c.HTML(http.StatusOK, "document_type_form.html", gin.H{
		"title": "Редагувати тип документа",
		"dt":    dt,
	})
}

// UpdateDocumentType handles the edit POST
func UpdateDocumentType(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var dt models.DocumentType
	if err := db.DB.First(&dt, id).Error; err != nil {
		c.String(http.StatusNotFound, "Тип не знайдено")
		return
	}
	dt.Name = strings.TrimSpace(c.PostForm("name"))
	db.DB.Save(&dt)
	c.Redirect(303, "/document-types")
}

// DeleteDocumentType deletes a document type
func DeleteDocumentType(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	db.DB.Delete(&models.DocumentType{}, id)
	c.Redirect(303, "/document-types")
}

// GetSettings renders the settings page
func GetSettings(c *gin.Context) {
	reminderDays := db.GetSetting("reminder_days", "30")
	reminderTime := db.GetSetting("reminder_time", "09:00")
	adminIDs := db.GetSetting("admin_chat_ids", "")
	msg := c.Query("msg")

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"title":        "Налаштування",
		"reminderDays": reminderDays,
		"reminderTime": reminderTime,
		"adminIDs":     adminIDs,
		"msg":          msg,
	})
}

// UpdateSettings handles the settings form POST
func UpdateSettings(c *gin.Context) {
	reminderDays := c.PostForm("reminder_days")
	reminderTime := c.PostForm("reminder_time")
	adminIDs := c.PostForm("admin_chat_ids")

	db.SetSetting("reminder_days", reminderDays)
	db.SetSetting("reminder_time", reminderTime)
	db.SetSetting("admin_chat_ids", adminIDs)

	c.Redirect(303, "/settings?msg=success")
}

// DownloadBackup allows downloading the SQLite database file
func DownloadBackup(c *gin.Context) {
	c.FileAttachment("fleet.db", "fleet_backup_"+time.Now().Format("20060102_150405")+".db")
}

// SendBackupToTelegram sends the database backup to Telegram
func SendBackupToTelegram(c *gin.Context) {
	chatIDStr := os.Getenv("BACKUP_CHAT_ID")
	if chatIDStr == "" {
		c.Redirect(303, "/settings?msg=error_no_chat_id")
		return
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		c.Redirect(303, "/settings?msg=error_invalid_chat_id")
		return
	}

	log.Printf("Sending backup to chatID: %d", chatID)

	// Log current DB counts before backup
	var carsCount, repairsCount, tireChangesCount, executorsCount, documentsCount, documentTypesCount, settingsCount int64
	db.DB.Model(&models.Car{}).Count(&carsCount)
	db.DB.Model(&models.Repair{}).Count(&repairsCount)
	db.DB.Model(&models.TireChange{}).Count(&tireChangesCount)
	db.DB.Model(&models.Executor{}).Count(&executorsCount)
	db.DB.Model(&models.Document{}).Count(&documentsCount)
	db.DB.Model(&models.DocumentType{}).Count(&documentTypesCount)
	db.DB.Model(&models.Setting{}).Count(&settingsCount)
	log.Printf("DB counts before backup - Cars: %d, Repairs: %d, TireChanges: %d, Executors: %d, Documents: %d, DocumentTypes: %d, Settings: %d", carsCount, repairsCount, tireChangesCount, executorsCount, documentsCount, documentTypesCount, settingsCount)

	err = backup.PerformBackupAndSend(bot.TgBot, "fleet.db", chatID)
	if err != nil {
		log.Printf("Manual backup failed: %v", err)
		c.Redirect(303, "/settings?msg=error_backup")
		return
	}

	c.Redirect(303, "/settings?msg=backup_sent")
}
func ImportBackup(c *gin.Context) {
	log.Printf("Starting import backup")
	file, err := c.FormFile("backup_file")
	if err != nil {
		log.Printf("Error getting form file: %v", err)
		c.String(http.StatusBadRequest, "Помилка завантаження файлу")
		return
	}
	log.Printf("File received: %s, size: %d", file.Filename, file.Size)

	// Сохраняем временный файл в рабочей директории
	tempPath := "imported_fleet.db"
	log.Printf("Temp path: %s", tempPath)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		log.Printf("Error saving uploaded file: %v", err)
		c.String(http.StatusInternalServerError, "Помилка збереження тимчасового файлу")
		return
	}
	log.Printf("File saved to temp")

	// Check file size
	if stat, err := os.Stat(tempPath); err == nil {
		log.Printf("Saved file size: %d bytes", stat.Size())
	} else {
		log.Printf("Failed to stat file: %v", err)
	}

	// Compute MD5 hash
	hash, err := computeMD5(tempPath)
	if err != nil {
		log.Printf("Warning: failed to compute MD5 for imported file: %v", err)
	} else {
		log.Printf("Imported file MD5: %s", hash)
	}

	// Check if file is a valid SQLite database
	if isSQLite, err := isSQLiteFile(tempPath); err != nil {
		log.Printf("Warning: failed to check if file is SQLite: %v", err)
	} else if !isSQLite {
		log.Printf("Error: imported file is not a valid SQLite database")
		c.String(http.StatusBadRequest, "Завантажений файл не є коректною базою даних SQLite")
		return
	} else {
		log.Printf("Imported file is a valid SQLite database")
	}

	// Replace database
	if err := db.ReplaceDB(tempPath); err != nil {
		log.Printf("Error replacing DB: %v", err)
		c.String(http.StatusInternalServerError, "Помилка заміни бази даних: "+err.Error())
		return
	}
	log.Printf("DB replaced successfully")

	// Clean up temp file
	if err := os.Remove(tempPath); err != nil {
		log.Printf("Warning: failed to remove temp file %s: %v", tempPath, err)
	}

	c.Redirect(http.StatusSeeOther, "/settings?imported=true")
}

// SyncExecutors ensures all executors mentioned in repairs/tires are in the Executors table
func SyncExecutors(c *gin.Context) {
	var repairExecs []string
	db.DB.Model(&models.Repair{}).Distinct().Pluck("executor", &repairExecs)

	var tireExecs []string
	db.DB.Model(&models.TireChange{}).Distinct().Pluck("executor", &tireExecs)

	allExecs := make(map[string]bool)
	for _, e := range repairExecs {
		if e != "" {
			allExecs[e] = true
		}
	}
	for _, e := range tireExecs {
		if e != "" {
			allExecs[e] = true
		}
	}

	count := 0
	for name := range allExecs {
		var existing models.Executor
		err := db.DB.Where("name = ?", name).First(&existing).Error
		if err != nil { // record not found
			db.DB.Create(&models.Executor{Name: name})
			count++
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "added": count})
}

// ResetAndSync purges all data (cars, repairs, tires, documents, executors, document_types), then runs a fresh sync
func ResetAndSync(c *gin.Context) {
	// 1. Purge all data (Unscoped to truly remove from soft-delete)
	db.DB.Unscoped().Where("1 = 1").Delete(&models.Repair{})
	db.DB.Unscoped().Where("1 = 1").Delete(&models.TireChange{})
	db.DB.Unscoped().Where("1 = 1").Delete(&models.Document{})
	db.DB.Unscoped().Where("1 = 1").Delete(&models.Car{})
	db.DB.Unscoped().Where("1 = 1").Delete(&models.Executor{})
	db.DB.Unscoped().Where("1 = 1").Delete(&models.DocumentType{})

	// 2. Trigger sync
	if err := db.SyncSheets(); err != nil {
		c.String(http.StatusInternalServerError, "Помилка при синхронізації: %v", err)
		return
	}

	c.Redirect(303, "/settings?msg=success_reset")
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

func isSQLiteFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil {
		return false, err
	}
	if n < 16 {
		return false, nil
	}

	// SQLite header starts with "SQLite format 3"
	return string(header[:15]) == "SQLite format 3", nil
}
