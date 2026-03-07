package db

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"fleet-management/internal/models"

	"github.com/xuri/excelize/v2"
)

func ImportAll(dir string) error {
	path := dir + "/Документы.xlsx"
	if _, err := os.Stat(path); err != nil {
		log.Println("Import file Документы.xlsx not found, skipping local import")
		return nil
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		log.Printf("Error opening %s: %v", path, err)
		return err
	}
	defer f.Close()

	// 1. Import Cars
	ImportCarsSheet(f)

	// 2. Import Documents
	ImportDocumentsSheet(f)

	// 3. Import Repairs
	ImportRepairsSheet(f)

	// 4. Import Tires
	ImportTiresSheet(f)

	return nil
}

func ImportCarsSheet(f *excelize.File) {
	rows, err := f.GetRows("Список Авто")
	if err != nil {
		log.Println("Sheet 'Список Авто' not found")
		return
	}

	for i, row := range rows {
		if i == 0 || len(row) < 4 {
			continue
		}
		num := strings.ToUpper(strings.TrimSpace(row[0]))
		year, _ := strconv.Atoi(row[2])
		car := models.Car{
			Number: num,
			Model:  row[1],
			Year:   year,
			Owner:  row[3],
		}
		DB.FirstOrCreate(&car, models.Car{Number: num})
	}
}

func ImportDocumentsSheet(f *excelize.File) {
	rows, err := f.GetRows("Документи")
	if err != nil {
		log.Println("Sheet 'Документи' not found")
		return
	}

	for i, row := range rows {
		if i == 0 || len(row) < 4 {
			continue
		}
		carNum := strings.TrimSpace(row[0])
		if carNum == "" {
			continue
		}

		car, err := FindCarSmart(carNum)
		fullNum := carNum
		var carID uint
		if err == nil {
			fullNum = car.Number
			carID = car.ID
		}

		// Cleanup date strings in case of unexpected padding
		issueStr := strings.TrimSpace(row[2])
		expiryStr := strings.TrimSpace(row[3])

		issueDate, _ := time.Parse("02.01.2006", issueStr)
		expiryDate, _ := time.Parse("02.01.2006", expiryStr)

		doc := models.Document{
			CarID:      carID,
			CarNumber:  fullNum,
			Type:       row[1],
			IssueDate:  issueDate,
			ExpiryDate: expiryDate,
			Status:     "Active",
		}
		// Avoid duplicates based on car, type and expiry
		var existing models.Document
		if DB.Where("car_number = ? AND type = ? AND expiry_date = ?", fullNum, doc.Type, doc.ExpiryDate).First(&existing).Error != nil {
			DB.Create(&doc)
		}
	}
}

func ImportRepairsSheet(f *excelize.File) {
	rows, err := f.GetRows("Ремонти Сінтайл")
	if err != nil {
		log.Println("Sheet 'Ремонти Сінтайл' not found")
		return
	}

	for i, row := range rows {
		if i == 0 || len(row) < 6 {
			continue
		}
		dateStr := strings.TrimSpace(row[0])
		date, _ := time.Parse("02.01.2006", dateStr)
		carNum := strings.TrimSpace(row[1])
		car, err := FindCarSmart(carNum)
		fullNum := carNum
		var carID uint
		if err == nil {
			fullNum = car.Number
			carID = car.ID
		}

		mileage, _ := strconv.Atoi(row[2])
		priceStr := strings.ReplaceAll(row[5], " ", "")
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		price, _ := strconv.ParseFloat(priceStr, 64)

		description := row[3]
		executor := row[4]
		notes := ""
		if len(row) > 6 {
			notes = row[6]
		}

		repair := models.Repair{
			Date:        date,
			CarID:       carID,
			CarNumber:   fullNum,
			Mileage:     mileage,
			Description: description,
			Executor:    executor,
			Price:       price,
			Notes:       notes,
		}

		// Robust deduplication: fetch all for same car and date, then compare normalized descriptions in Go
		var existing []models.Repair
		DB.Where("car_number = ? AND date(date) = date(?)", fullNum, date).Find(&existing)

		isDuplicate := false
		normDesc := NormalizeDescription(description)
		for _, e := range existing {
			if NormalizeDescription(e.Description) == normDesc {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			DB.Create(&repair)
		}
	}
}

func ImportTiresSheet(f *excelize.File) {
	rows, err := f.GetRows("Заміна Резини")
	if err != nil {
		log.Println("Sheet 'Заміна Резини' not found")
		return
	}

	for i, row := range rows {
		if i == 0 {
			continue
		} // Columns: [Date(0), Number(1), Vehicle(2), Mileage(3), Description(4), Executor(5), Price(6), Notes(7)]
		dateStr := strings.TrimSpace(row[0])
		date, _ := time.Parse("02.01.2006", dateStr)
		carNum := strings.TrimSpace(row[1])
		car, err := FindCarSmart(carNum)
		fullNum := carNum
		var carID uint
		if err == nil {
			fullNum = car.Number
			carID = car.ID
		}

		mileage, _ := strconv.Atoi(row[3])
		executor := ""
		if len(row) > 5 {
			executor = row[5]
		}
		priceStr := ""
		if len(row) > 6 {
			priceStr = strings.ReplaceAll(row[6], " ", "")
			priceStr = strings.ReplaceAll(priceStr, ",", ".")
		}
		price, _ := strconv.ParseFloat(priceStr, 64)
		notes := ""
		if len(row) > 7 {
			notes = row[7]
		}

		tire := models.TireChange{
			Date:        date,
			CarID:       carID,
			CarNumber:   fullNum,
			Mileage:     mileage,
			Description: row[4],
			Executor:    executor,
			Price:       price,
			Notes:       notes,
		}

		// Robust deduplication: fetch all for same car and date, then compare normalized descriptions in Go
		var existing []models.TireChange
		DB.Where("car_number = ? AND date(date) = date(?)", fullNum, date).Find(&existing)

		isDuplicate := false
		normDesc := NormalizeDescription(tire.Description)
		for _, e := range existing {
			if NormalizeDescription(e.Description) == normDesc {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			DB.Create(&tire)
		}
	}
}
