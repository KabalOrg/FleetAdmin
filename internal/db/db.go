package db

import (
	"fmt" // Added for fmt.Errorf
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"fleet-management/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Car{},
		&models.Document{},
		&models.Repair{},
		&models.TireChange{},
		&models.Executor{},
		&models.DocumentType{},
		&models.Setting{},
	)
	if err != nil {
		return nil, err
	}

	DB = db

	// Seed default document types if none exist
	var dtCount int64
	db.Model(&models.DocumentType{}).Count(&dtCount)
	if dtCount == 0 {
		defaults := []models.DocumentType{
			{Name: "Страхування"},
			{Name: "Техпаспорт"},
			{Name: "Медична довідка"},
			{Name: "Інше"},
		}
		db.Create(&defaults)
	}

	return db, nil
}

// FindCarSmart searches for a car by full number or a suffix (e.g., last 4 digits).
func FindCarSmart(input string) (*models.Car, error) {
	var car models.Car
	input = strings.ToUpper(strings.TrimSpace(input))
	if err := DB.Where("number = ?", input).First(&car).Error; err == nil {
		return &car, nil
	}
	// Try partial match
	if err := DB.Where("number LIKE ?", "%"+input+"%").First(&car).Error; err == nil {
		return &car, nil
	}
	return nil, fmt.Errorf("car not found")
}

// NormalizeDescription removes multiple spaces, punctuation and lowercases the string
func NormalizeDescription(s string) string {
	s = strings.ToLower(s)
	// Replace punctuation with spaces
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "(", " ", ")", " ", "-", " ", "_", " ")
	s = replacer.Replace(s)
	// Collapse multiple spaces
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// GetSetting returns the value of a setting by key.
func GetSetting(key string, defaultValue string) string {
	var s models.Setting
	if err := DB.Where("key = ?", key).First(&s).Error; err != nil {
		return defaultValue
	}
	return s.Value
}

// SetSetting saves or updates a setting.
func SetSetting(key, value string) error {
	var s models.Setting
	err := DB.Where("key = ?", key).First(&s).Error
	if err != nil {
		s = models.Setting{Key: key, Value: value}
		return DB.Create(&s).Error
	}
	s.Value = value
	return DB.Save(&s).Error
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ReplaceDB replaces the current database file with a new one
func ReplaceDB(newPath string) error {
	log.Printf("Starting ReplaceDB with newPath: %s", newPath)

	// 1. Close current connection
	if err := CloseDB(); err != nil {
		log.Printf("Failed to close DB: %v", err)
		return fmt.Errorf("failed to close database: %v", err)
	}
	log.Printf("DB closed")

	// 2. Backup current (optional but safe)
	_ = os.Rename("fleet.db", "fleet.db.bak")
	log.Printf("Old DB backed up")

	// 3. Move new file to fleet.db
	source, err := os.Open(newPath)
	if err != nil {
		log.Printf("Failed to open source: %v", err)
		return err
	}
	defer source.Close()

	destination, err := os.Create("fleet.db")
	if err != nil {
		log.Printf("Failed to create destination: %v", err)
		return err
	}
	defer destination.Close()

	copied, err := io.Copy(destination, source)
	if err != nil {
		log.Printf("Failed to copy: %v", err)
		return err
	}
	log.Printf("Copied %d bytes", copied)

	// 4. Re-initialize
	_, err = InitDB("fleet.db")
	if err != nil {
		log.Printf("Failed to InitDB: %v", err)
		return err
	}
	log.Printf("DB re-initialized successfully")

	// Log counts to verify data restoration
	var carsCount, repairsCount, tireChangesCount, executorsCount, documentsCount, documentTypesCount, settingsCount int64
	DB.Model(&models.Car{}).Count(&carsCount)
	DB.Model(&models.Repair{}).Count(&repairsCount)
	DB.Model(&models.TireChange{}).Count(&tireChangesCount)
	DB.Model(&models.Executor{}).Count(&executorsCount)
	DB.Model(&models.Document{}).Count(&documentsCount)
	DB.Model(&models.DocumentType{}).Count(&documentTypesCount)
	DB.Model(&models.Setting{}).Count(&settingsCount)
	log.Printf("Counts after replace - Cars: %d, Repairs: %d, TireChanges: %d, Executors: %d, Documents: %d, DocumentTypes: %d, Settings: %d", carsCount, repairsCount, tireChangesCount, executorsCount, documentsCount, documentTypesCount, settingsCount)

	return nil
}

// EvaluatePrice calculates the total from a simple expression like "2500+1500"
func EvaluatePrice(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, nil
	}
	// Support simple addition for now
	if strings.Contains(input, "+") {
		parts := strings.Split(input, "+")
		var total float64
		for _, part := range parts {
			val, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", part)
			}
			total += val
		}
		return total, nil
	}
	// Fallback to regular parse
	return strconv.ParseFloat(input, 64)
}
