# Fleet Management System

Система управління автопарком з веб-інтерфейсом та інтеграцією з Telegram для нагадувань.

## Особливості

- **Управління автопарком**: Додавання, редагування та видалення автомобілів
- **Документи**: Відстеження термінів дії документів (страхування, техпаспорт тощо)
- **Ремонти та шини**: Журнал ремонтів та заміни шин
- **Telegram бот**: Автоматичні нагадування про закінчення термінів документів
- **Синхронізація з Google Sheets**: Імпорт даних з електронних таблиць
- **Бекап та відновлення**: Автоматичне створення бекапів у Telegram та ручне імпортування
- **Веб-інтерфейс**: Зручний інтерфейс для управління даними з Tailwind CSS

## Встановлення

Найпростіший спосіб - використовувати готовий `.exe` файл. Вам більше не потрібно створювати та заповнювати `.env` вручну або копіювати папки з шаблонами.

### Спосіб 1: Швидкий старт (Вбудований Web-установник)

1. Завантажте готовий `fleet-app.exe`.
2. Запустіть файл. При першому запуску відкриється консоль з повідомленням про запуск веб-установника.
3. Відкрийте у браузері `http://localhost:8080`.
4. Заповніть форму (вкажіть Telegram Bot Token, пароль і опціонально завантажте ваш `fleet.db` бекап).
5. Натисніть "Зберегти та Продовжити". Програма автоматично створить `.env` і запустить систему.

### Спосіб 2: Збірка з вихідного коду

```bash
git clone https://github.com/your-username/fleet-management.git
cd fleet-management
go mod download
go build -o fleet-app.exe cmd/main.go
./fleet-app.exe

Сервер буде доступний за адресою `http://localhost:8080`

#### З Docker

```bash
docker-compose up -d
```

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

1. Підготуйте Google Sheet з даними (таблиці: Cars, Repairs, Tires)
2. Налаштуйте доступ через Google API (service account JSON)
3. Використовуйте функцію синхронізації в налаштуваннях

### Бекап та відновлення

- **Автоматичний бекап**: Щодня о 01:00 створюється бекап та відправляється в Telegram
- **Ручний бекап**: В налаштуваннях можна створити та завантажити бекап
- **Відновлення**: Завантажте файл бекапа через веб-інтерфейс для відновлення даних

## API Endpoints

### Автомобілі
- `GET /fleet` - Список автомобілів
- `GET /fleet/detail/:id` - Деталі автомобіля
- `POST /cars/add` - Додати автомобіль
- `POST /cars/edit/:id` - Редагувати автомобіль
- `POST /cars/delete/:id` - Видалити автомобіль

### Документи
- `GET /documents` - Список документів
- `POST /documents/add` - Додати документ
- `POST /documents/edit/:id` - Редагувати документ
- `POST /documents/delete/:id` - Видалити документ

### Ремонти
- `GET /repairs` - Список ремонтів
- `POST /repairs/add` - Додати ремонт
- `POST /repairs/edit/:id` - Редагувати ремонт
- `POST /repairs/delete/:id` - Видалити ремонт

### Шини
- `GET /tires` - Список заміни шин
- `POST /tires/add` - Додати заміну шин
- `POST /tires/edit/:id` - Редагувати заміну шин
- `POST /tires/delete/:id` - Видалити заміну шин

### Налаштування
- `GET /settings` - Налаштування системи
- `POST /settings/update` - Оновити налаштування
- `POST /settings/send-backup` - Створити бекап
- `POST /settings/import` - Імпортувати бекап

## Структура проекту

```
.
├── cmd/
│   └── main.go                 # Точка входу
├── internal/
│   ├── backup/                 # Логіка бекапів
│   │   └── backup.go
│   ├── bot/                    # Telegram бот
│   │   ├── bot.go
│   │   ├── reminders.go
│   ├── db/                     # Робота з базою даних
│   │   ├── db.go
│   │   ├── google_sync.go
│   │   ├── import.go
│   ├── handlers/               # HTTP обробники
│   │   └── handlers.go
│   └── models/                 # Моделі даних
│       └── models.go
├── static/                     # Статичні файли (CSS, JS)
├── templates/                  # HTML шаблони
├── docker-compose.yml          # Docker Compose
├── Dockerfile                  # Docker образ
├── go.mod                      # Go модулі
├── go.sum
├── .env.example                # Приклад змінних середовища
├── .gitignore
└── README.md
```

## Розробка

### Запуск у режимі розробки

```bash
go run cmd/main.go
```

### Тестування

```bash
go test ./...
```

### Збірка

```bash
go build -o fleet-app.exe ./cmd
```

## Docker

### Збірка образу

```bash
docker build -t fleet-management .
```

### Запуск контейнера

```bash
docker run -p 8080:8080 --env-file .env fleet-management
```

## Внесок

1. Форкніть репозиторій
2. Створіть гілку для вашої функції (`git checkout -b feature/AmazingFeature`)
3. Зробіть коміти (`git commit -m 'Add some AmazingFeature'`)
4. Запуште гілку (`git push origin feature/AmazingFeature`)
5. Відкрийте Pull Request

## Ліцензія

Цей проект ліцензовано під MIT License - дивіться файл [LICENSE](LICENSE) для деталей.

## Підтримка

Якщо у вас є питання або проблеми, створіть issue в репозиторії або зв'яжіться з розробником.

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

## 🍕 Підтримати проект (Donate)

Якщо ця система допомогла вашому бізнесу або провайдеру пережити відключення світла, ви можете підтримати розвиток проекту:

<a href="https://donatello.to/kabal_org" target="_blank">
  <img src="https://img.shields.io/badge/Підтримати_на-Donatello-FF5722?style=for-the-badge" alt="Donatello">
</a>


<a href="https://send.monobank.ua/jar/Abc4m6jPBC" target="_blank">
  <img src="https://img.shields.io/badge/Прямий_донат-Monobank-000000?style=for-the-badge" alt="Monobank">
</a>
