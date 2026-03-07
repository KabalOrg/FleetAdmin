# Fleet Management System

Система управління автопарком з веб-інтерфейсом та інтеграцією з Telegram для нагадувань.

## Особливості

- **Управління автопарком**: Додавання, редагування та видалення автомобілів
- **Документи**: Відстеження термінів дії документів (страхування, техпаспорт тощо)
- **Ремонти та шини**: Журнал ремонтів та заміни шин
- **Telegram бот**: Автоматичні нагадування про закінчення термінів документів
- **Синхронізація з Google Sheets**: Імпорт даних з електронних таблиць
- **Веб-інтерфейс**: Зручний інтерфейс для управління даними

## Встановлення

### Передумови

- Go 1.19+
- SQLite3

### Клонування репозиторію

```bash
git clone https://github.com/your-username/fleet-management.git
cd fleet-management
```

### Встановлення залежностей

```bash
go mod download
```

### Налаштування змінних середовища

Створіть файл `.env` в корені проекту:

```env
# Telegram Bot Token (отримайте від @BotFather)
TELEGRAM_BOT_TOKEN=your_telegram_bot_token

# Пароль адміністратора для доступу до налаштувань
ADMIN_PASSWORD=your_secure_password

# Порт сервера (необов'язково, за замовчуванням 8080)
PORT=8080
```

### Запуск

```bash
go run cmd/main.go
```

Сервер буде доступний за адресою `http://localhost:8080`

## Використання

### Веб-інтерфейс

1. **Головна сторінка**: Огляд системи
2. **Автопарк**: Перегляд та управління автомобілями
3. **Документи**: Управління документами автомобілів
4. **Ремонти**: Журнал ремонтів
5. **Шини**: Журнал заміни шин
6. **Налаштування**: Конфігурація системи (потребує аутентифікації)

### Налаштування Telegram бота

1. Створіть бота через @BotFather в Telegram
2. Отримайте токен та додайте в `.env`
3. В налаштуваннях веб-інтерфейсу вкажіть Chat ID для отримання нагадувань
4. Налаштуйте час та дні для нагадувань

### Синхронізація з Google Sheets

1. Підготуйте Google Sheet з даними
2. Налаштуйте доступ через Google API
3. Використовуйте функцію синхронізації в налаштуваннях

## Структура проекту

```
.
├── cmd/
│   └── main.go                 # Точка входу
├── internal/
│   ├── bot/                    # Telegram бот
│   │   ├── bot.go
│   │   ├── reminders.go
│   │   └── ...
│   ├── db/                     # Робота з базою даних
│   ├── handlers/               # HTTP обробники
│   └── models/                 # Моделі даних
├── templates/                  # HTML шаблони
├── static/                     # Статичні файли
├── go.mod
├── go.sum
└── README.md
```

## API Endpoints

### Публічні маршрути

- `GET /` - Головна сторінка
- `GET /fleet` - Список автомобілів
- `GET /documents` - Список документів
- `GET /repairs` - Список ремонтів
- `GET /tires` - Список заміни шин
- `GET /login` - Сторінка входу
- `POST /login` - Авторизація

### Захищені маршрути (потребують аутентифікації)

- `GET /settings` - Налаштування системи
- `POST /settings/update` - Оновлення налаштувань
- `GET /settings/backup` - Завантаження бекапу БД
- `POST /settings/import` - Імпорт БД

## Скриншоти інтерфейсу
<img width="1916" height="933" alt="gitfleet" src="https://github.com/user-attachments/assets/c1be3854-7e6c-4d10-8d9a-252d5979961a" />
<img width="1916" height="933" alt="gitfleet2" src="https://github.com/user-attachments/assets/0a94bae6-6d5e-4fef-b868-cb541140121b" />
<img width="1916" height="933" alt="gitfleet3" src="https://github.com/user-attachments/assets/70f39c1e-91a4-4798-b582-4c4a42359ff3" />
<img width="1916" height="933" alt="gitfleet4" src="https://github.com/user-attachments/assets/2ada0b93-3b93-46ba-b97c-2f57b77cadd5" />
<img width="1916" height="933" alt="gitfleet5" src="https://github.com/user-attachments/assets/f0c81690-ba38-42ff-a22f-b987eccf70a2" />



## Розробка

### Додавання нових функцій

1. Створіть модель в `internal/models/`
2. Додайте обробники в `internal/handlers/`
3. Створіть шаблони в `templates/`
4. Додайте маршрути в `cmd/main.go`

### Міграції бази даних

Система використовує GORM auto-migration. Нові поля додаються автоматично при запуску.

## Ліцензія

MIT License

## Підтримка

Для питань та пропозицій створюйте issues в репозиторії.
