package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"fleet-management/internal/models"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// SyncSheets coordinates the synchronization from Google Sheets
func SyncSheets() error {
	ctx := context.Background()
	credFile := "credentials.json"

	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		return fmt.Errorf("credentials.json not found")
	}

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credFile))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	// Spreadsheet IDs - normally these should be in .env
	// I'll look for environment variables or use placeholders
	spreadsheetID := os.Getenv("GOOGLE_SHEET_ID")
	if spreadsheetID == "" {
		log.Println("GOOGLE_SHEET_ID not set in .env. Skipping Sheets sync.")
		return nil
	}

	// If a full URL was provided, extract the ID
	if strings.Contains(spreadsheetID, "/d/") {
		parts := strings.Split(spreadsheetID, "/d/")
		if len(parts) > 1 {
			spreadsheetID = strings.Split(parts[1], "/")[0]
		}
	}

	// Sync Cars
	if err := syncCars(srv, spreadsheetID); err != nil {
		log.Printf("Error syncing cars: %v", err)
	}

	// Sync Repairs
	if err := syncRepairs(srv, spreadsheetID); err != nil {
		log.Printf("Error syncing repairs: %v", err)
	}

	// Sync Tires
	if err := syncTires(srv, spreadsheetID); err != nil {
		log.Printf("Error syncing tires: %v", err)
	}

	return nil
}

func syncCars(srv *sheets.Service, id string) error {
	readRange := "Cars!A2:E"
	resp, err := srv.Spreadsheets.Values.Get(id, readRange).Do()
	if err != nil {
		return err
	}

	for _, row := range resp.Values {
		if len(row) < 2 {
			continue
		}
		num := strings.ToUpper(fmt.Sprintf("%v", row[0]))
		model := fmt.Sprintf("%v", row[1])
		year, _ := strconv.Atoi(fmt.Sprintf("%v", row[2]))
		owner := ""
		if len(row) > 3 {
			owner = fmt.Sprintf("%v", row[3])
		}

		car := models.Car{
			Number: num,
			Model:  model,
			Year:   year,
			Owner:  owner,
		}
		DB.FirstOrCreate(&car, models.Car{Number: num})
	}
	return nil
}

func syncRepairs(srv *sheets.Service, id string) error {
	readRange := "Repairs!A2:G"
	resp, err := srv.Spreadsheets.Values.Get(id, readRange).Do()
	if err != nil {
		return err
	}

	for _, row := range resp.Values {
		if len(row) < 6 {
			continue
		}
		date, _ := time.Parse("02.01.2006", fmt.Sprintf("%v", row[0]))
		carNum := fmt.Sprintf("%v", row[1])
		mileage, _ := strconv.Atoi(fmt.Sprintf("%v", row[2]))
		description := fmt.Sprintf("%v", row[3])
		executor := fmt.Sprintf("%v", row[4])
		price, _ := strconv.ParseFloat(fmt.Sprintf("%v", row[5]), 64)
		notes := ""
		if len(row) > 6 {
			notes = fmt.Sprintf("%v", row[6])
		}

		car, err := FindCarSmart(carNum)
		fullNum := carNum
		var carID uint
		if err == nil {
			fullNum = car.Number
			carID = car.ID
		}

		// Improved de-duplication: fetch all for same car and date, then compare normalized descriptions in Go
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
			DB.Create(&models.Repair{
				Date:        date,
				CarID:       carID,
				CarNumber:   fullNum,
				Mileage:     mileage,
				Description: description,
				Executor:    executor,
				Price:       price,
				Notes:       notes,
			})
		}
	}
	return nil
}

func syncTires(srv *sheets.Service, id string) error {
	readRange := "Tires!A2:E"
	resp, err := srv.Spreadsheets.Values.Get(id, readRange).Do()
	if err != nil {
		return err
	}

	for _, row := range resp.Values {
		if len(row) < 5 {
			continue
		}
		date, _ := time.Parse("02.01.2006", fmt.Sprintf("%v", row[0]))
		carNum := fmt.Sprintf("%v", row[1])
		mileage, _ := strconv.Atoi(fmt.Sprintf("%v", row[2]))
		description := fmt.Sprintf("%v", row[3])
		price, _ := strconv.ParseFloat(fmt.Sprintf("%v", row[4]), 64)

		car, err := FindCarSmart(carNum)
		fullNum := carNum
		var carID uint
		if err == nil {
			fullNum = car.Number
			carID = car.ID
		}

		var existing []models.TireChange
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
			DB.Create(&models.TireChange{
				Date:        date,
				CarID:       carID,
				CarNumber:   fullNum,
				Mileage:     mileage,
				Description: description,
				Price:       price,
			})
		}
	}
	return nil
}
