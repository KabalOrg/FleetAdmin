package installer

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed installer.html
var installerHTML string

// Run starts the web installer if necessary parameters are missing.
// It blocks until the user submits the form and a valid .env is created.
func Run() error {
	mux := http.NewServeMux()
	
	// Create a channel to signal completion
	done := make(chan struct{})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(installerHTML))
			return
		}

		if r.Method == http.MethodPost {
			// parse form (max 50 MB for DB backups)
			err := r.ParseMultipartForm(50 << 20)
			if err != nil {
				http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
				return
			}
			
			token := r.FormValue("TELEGRAM_BOT_TOKEN")
			port := r.FormValue("PORT")
			if port == "" {
				port = "8080"
			}
			adminPassword := r.FormValue("ADMIN_PASSWORD")
			if adminPassword == "" {
				adminPassword = "admin"
			}
			
			authorizedUsers := r.FormValue("AUTHORIZED_USER_IDS")
			backupChatID := r.FormValue("BACKUP_CHAT_ID")
			
			// Handle DB upload
			file, _, err := r.FormFile("databaseBackup")
			if err == nil {
				defer file.Close()
				dbFile, err := os.Create("fleet.db")
				if err != nil {
					log.Printf("Failed to create fleet.db: %v", err)
				} else {
					defer dbFile.Close()
					io.Copy(dbFile, file)
					log.Println("Imported database from installer")
				}
			}

			// Generate .env
			envContent := fmt.Sprintf(`TELEGRAM_BOT_TOKEN=%s
PORT=%s
ADMIN_PASSWORD=%s
AUTHORIZED_USER_IDS=%s
BACKUP_CHAT_ID=%s
GIN_MODE=release
`, token, port, adminPassword, authorizedUsers, backupChatID)
			
			err = os.WriteFile(".env", []byte(envContent), 0644)
			if err != nil {
				http.Error(w, "Failed to write .env: "+err.Error(), http.StatusInternalServerError)
				return
			}
			
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(`
<html><body style="font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; background-color: #f3f4f6; text-align: center;">
<div>
	<h2>Установка успешно завершена!</h2>
	<p>.env файл сохранен. Приложение сейчас запустится...</p>
	<p>Вас перенаправят в панель управления через 3 секунды.</p>
</div>
<script>
	setTimeout(function() {
		window.location.href = "/fleet";
	}, 3000);
</script>
</body></html>
`))
			
			// Signal completion asynchronously so the response can be flushed
			go func() {
				time.Sleep(1 * time.Second) // wait for response to be sent to user
				close(done)
			}()
		}
	})

	server := &http.Server{Addr: ":8080", Handler: mux}

	go func() {
		<-done
		log.Println("Установщик завершил работу, запускаем основное приложение...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	fmt.Println("\n=================================================================")
	fmt.Println("🚀 ВАЖНО: ОТСУТСТВУЕТ ФАЙЛ НАСТРОЕК (.env) ИЛИ ТОКЕН БОТА")
	fmt.Println("👉 Запущен Web-установщик!")
	fmt.Println("🌐 Пожалуйста, откройте в браузере: http://localhost:8080")
	fmt.Println("=================================================================")

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	
	return nil
}
