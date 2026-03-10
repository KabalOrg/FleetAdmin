package main

import (
	"log"
	"os"
	"strconv"

	"fleet-management/internal/backup"
	"fleet-management/internal/bot"
	"fleet-management/internal/db"
	"fleet-management/internal/handlers"
	"fleet-management/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	} else {
		log.Printf(".env file loaded successfully")
	}

	// Initialize Database
	dbConn, err := db.InitDB("fleet.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Seed some test data if fleet is empty
	var count int64
	dbConn.Model(&models.Car{}).Count(&count)
	if count == 0 {
		dbConn.Create(&models.Car{Number: "BI5565CK", Model: "Skoda Octavia", Year: 2020, Owner: "Kabal"})
	}

	// Auto-import local files if they exist
	_ = db.ImportAll(".")

	// Start Telegram Bot in goroutine
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken != "" {
		tgBot := bot.StartBot(botToken)
		bot.TgBot = tgBot
		bot.StartReminderWorker(tgBot)

		// Start backup worker
		backupChatIDStr := os.Getenv("BACKUP_CHAT_ID")
		if backupChatIDStr != "" {
			if backupChatID, err := strconv.ParseInt(backupChatIDStr, 10, 64); err == nil {
				go backup.StartBackupWorker(tgBot, "fleet.db", backupChatID)
			} else {
				log.Printf("Failed to parse BACKUP_CHAT_ID: %v", err)
			}
		} else {
			log.Println("WARNING: BACKUP_CHAT_ID not set, backup worker will not start")
		}
	} else {
		log.Println("WARNING: TELEGRAM_BOT_TOKEN not set, bot will not start")
	}

	r := gin.Default()

	// Setup sessions
	store := cookie.NewStore([]byte("secret-key-change-in-production"))
	r.Use(sessions.Sessions("admin-session", store))

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

	// Auth middleware
	authRequired := func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("authenticated") != true {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}
		c.Next()
	}

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/fleet")
	})
	r.GET("/repairs", handlers.GetRepairs)
	r.GET("/repairs/add", handlers.AddRepair)
	r.POST("/repairs/add", handlers.CreateRepair)
	r.GET("/repairs/edit/:id", handlers.EditRepair)
	r.POST("/repairs/edit/:id", handlers.UpdateRepair)
	r.POST("/repairs/delete/:id", handlers.DeleteRepair)
	r.GET("/tires", handlers.GetTires)
	r.GET("/tires/add", handlers.AddTire)
	r.POST("/tires/add", handlers.CreateTire)
	r.GET("/tires/edit/:id", handlers.EditTire)
	r.POST("/tires/edit/:id", handlers.UpdateTire)
	r.POST("/tires/delete/:id", handlers.DeleteTire)
	r.GET("/import", func(c *gin.Context) {
		db.ImportAll(".")
		c.Redirect(303, "/fleet")
	})
	r.GET("/sync", func(c *gin.Context) {
		if err := db.SyncSheets(); err != nil {
			log.Printf("Sync error: %v", err)
		}
		c.Redirect(303, "/fleet")
	})
	r.GET("/fleet", handlers.GetFleet)
	r.GET("/fleet/add", handlers.AddCar)
	r.POST("/fleet/add", handlers.CreateCar)
	r.GET("/fleet/edit/:id", handlers.EditCar)
	r.POST("/fleet/edit/:id", handlers.UpdateCar)
	r.POST("/fleet/delete/:id", handlers.DeleteCar)
	r.GET("/fleet/detail/:id", handlers.GetCarDetail)
	r.GET("/api/search", handlers.SearchCars)
	r.GET("/documents", handlers.GetDocuments)
	r.GET("/documents/add", handlers.AddDocument)
	r.POST("/documents/add", handlers.CreateDocument)
	r.GET("/documents/edit/:id", handlers.EditDocument)
	r.POST("/documents/edit/:id", handlers.UpdateDocument)
	r.POST("/documents/delete/:id", handlers.DeleteDocument)

	r.GET("/executors", handlers.GetExecutors)
	r.GET("/executors/add", handlers.AddExecutor)
	r.POST("/executors/add", handlers.CreateExecutor)
	r.GET("/executors/edit/:id", handlers.EditExecutor)
	r.POST("/executors/edit/:id", handlers.UpdateExecutor)
	r.POST("/executors/delete/:id", handlers.DeleteExecutor)

	r.GET("/document-types", handlers.GetDocumentTypes)
	r.GET("/document-types/add", handlers.AddDocumentType)
	r.POST("/document-types/add", handlers.CreateDocumentType)
	r.GET("/document-types/edit/:id", handlers.EditDocumentType)
	r.POST("/document-types/edit/:id", handlers.UpdateDocumentType)
	r.POST("/document-types/delete/:id", handlers.DeleteDocumentType)

	// Auth routes
	r.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("authenticated") == true {
			c.Redirect(302, "/settings")
			return
		}
		errorMsg := c.Query("error")
		c.HTML(200, "login.html", gin.H{"error": errorMsg})
	})
	r.POST("/login", func(c *gin.Context) {
		password := c.PostForm("password")
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin" // default for development
		}
		log.Printf("Login attempt: input password length %d, admin password: %s", len(password), adminPassword)
		if password == adminPassword {
			session := sessions.Default(c)
			session.Set("authenticated", true)
			session.Save()
			log.Printf("Login successful, redirecting to /settings")
			c.Redirect(302, "/settings")
		} else {
			log.Printf("Login failed: wrong password")
			c.Redirect(302, "/login?error=Невірний пароль")
		}
	})
	r.POST("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()
		c.Redirect(302, "/")
	})

	// Protected routes
	r.GET("/settings", authRequired, handlers.GetSettings)
	r.POST("/settings/update", authRequired, handlers.UpdateSettings)
	r.GET("/settings/backup", authRequired, handlers.DownloadBackup)
	r.POST("/settings/send-backup", authRequired, handlers.SendBackupToTelegram)
	r.POST("/settings/import", authRequired, handlers.ImportBackup)
	r.GET("/executors/sync", authRequired, handlers.SyncExecutors)
	r.GET("/repairs/reset", authRequired, handlers.ResetAndSync)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to run server:", err)
	}
}
