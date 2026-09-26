# PixelLife Tracker: что сделано и как это работает

## 1. Что это за проект

PixelLife Tracker - личный life tracker по ТЗ из `PixelLife_Tracker_Prompt.docx`.

Стек:

- Backend: Go + Fiber.
- Database: SQLite.
- Frontend: React + Vite.
- Icons: lucide-react.
- PWA: web manifest + service worker.
- Deployment: Docker Compose, один контейнер с backend и собранным frontend.
- Telegram: официальный Bot API через `net/http`, без тяжёлой дополнительной библиотеки.
- AI: Gemini API через асинхронный weekly assistant.

Главная идея: пользователь видит dashboard с балансом сфер жизни, а все записи сохраняются в SQLite и доступны из web-интерфейса и Telegram.

## 2. Что было реализовано

### Frontend

В `frontend/src/app.jsx` реализованы рабочие разделы:

- Dashboard.
- Учёба.
- Pomodoro.
- Полка книг и сериалов.
- Сон.
- Спорт.
- Стрики хобби.
- F1-события.
- История учебных сессий.
- Ввод результатов SAT, IELTS и NISH.
- Заметки к учебным сессиям и media items.
- Мобильная нижняя навигация.
- Offline snapshot через localStorage.
- Реальный индикатор подключения backend через `/api/health`.

В `frontend/src/functional.css` находятся стили форм, списков, карточек и мобильной раскладки.

В `frontend/src/styles.css` находится основная тема. Pixel-стиль сделан не только сменой шрифта:

- фоновая сетка с шагом 8px;
- квадратные кнопки и границы;
- жёсткие пиксельные тени без больших скруглений;
- bitmap-шрифт `Press Start 2P` для заголовков;
- маленькие игровые labels и status markers;
- цветовые состояния для учебы, спорта, сна и хобби;
- responsive layout для телефона.

### Dashboard и radar chart

Dashboard получает данные через:

```text
GET /api/dashboard
```

Сервер считает значения по последним семи дням:

- study: минуты учебы;
- sport: дни с завершённой тренировкой;
- hobby: завершённые hobby sessions;
- routine: дни со сном от 7 часов.

Если API доступен, цифры radar берутся из SQLite. Если API временно недоступен, frontend показывает последний сохранённый snapshot, но верхний индикатор честно показывает offline-состояние.

### Pomodoro

Pomodoro находится на Dashboard.

Состояние хранится в `localStorage` в ключе `pixellife_pomodoro`:

- выбранная категория;
- остаток времени;
- статус running/paused;
- абсолютное время окончания `endAt`.

Важный момент: таймер не полагается только на количество секунд. При reload он пересчитывает остаток как:

```text
endAt - Date.now()
```

Поэтому закрытие или обновление страницы не сбрасывает активный Pomodoro обратно на 25:00.

Учебная сессия отправляется в backend только после полного завершения таймера:

```text
POST /api/study/sessions
```

Пауза не создаёт ложную учебную сессию.

### Учёба

Frontend использует API:

```text
GET  /api/study/sessions
POST /api/study/sessions
GET  /api/study/scores/:category
POST /api/study/scores
```

Форма сессии содержит:

- категорию SAT / IELTS / NISH;
- длительность;
- оценку качества;
- заметку.

Отдельная форма добавляет результат. На экране отображаются последние баллы и история сессий, чтобы со временем видеть связь `время vs результат`.

### Полка

Frontend использует API `/api/media`.

Для каждого элемента доступны:

- тип: книга или сериал;
- название;
- статус: planned, in_progress или done;
- URL обложки;
- сезон и серия для сериалов;
- заметки.

### Сон, спорт, хобби и F1

Используемые endpoints:

```text
POST /api/sleep/records
POST /api/sport/sessions
GET  /api/hobby/streaks
POST /api/hobby/streaks
POST /api/hobby/streaks/:id/complete
POST /api/f1/events
```

Можно записать:

- количество часов сна;
- время отбоя и подъёма;
- домашнюю тренировку и комментарий;
- хобби-стрик и ежедневное выполнение;
- важную дату F1 weekend.

## 3. Backend

Основная точка запуска:

```text
backend/cmd/server/main.go
```

Backend:

1. Загружает `.env` через `godotenv`.
2. Открывает SQLite.
3. Создаёт таблицы и индексы при старте.
4. Создаёт service и handler для каждого раздела.
5. Запускает Telegram polling, если токен валиден.
6. Запускает асинхронный Gemini assistant, если есть ключ.
7. Раздаёт собранный frontend из `./public`.
8. Слушает порт из `PORT`.

Health check:

```text
GET /api/health
```

Пример успешного ответа:

```json
{
  "status": "ok",
  "telegram_configured": true,
  "gemini_configured": true,
  "personal_auth": true
}
```

## 4. Авторизация

Для приватного личного трекера используется:

```env
PERSONAL_AUTH=true
```

В этом режиме frontend создаёт локального пользователя с тестовым Telegram ID и получает JWT. Это удобно для одного владельца приватного VPS.

Для публичного приложения нужно поставить:

```env
PERSONAL_AUTH=false
```

и использовать настоящий Telegram WebApp `init_data`. В production backend уже умеет проверять подпись Telegram init data, но frontend должен получать настоящие данные из Telegram WebApp, а не тестовый ID.

JWT хранится в browser localStorage под ключом `pixellife_token`. Если API отвечает 401, frontend очищает старый token, получает новый и повторяет авторизацию.

## 5. Telegram bot

Файл реализации:

```text
backend/internal/bot/bot.go
```

Бот работает через long polling. Публичный webhook для базовой работы не нужен.

Поддерживаемые команды:

```text
/start
/sleep 7.5
/sport
/shelf Dune
```

Команды пишут данные в ту же SQLite базу, что и web-приложение.

При старте backend сначала вызывает Telegram `getMe`. В логах будет одно из сообщений:

```text
Telegram bot connected
```

или:

```text
Telegram disabled: ...
```

Если `getMe` возвращает 401, токен нужно перевыпустить через BotFather. Старый токен нельзя оставлять в использовании.

## 6. Gemini assistant

Файл реализации:

```text
backend/internal/assistant/assistant.go
```

Assistant работает в отдельной goroutine и не блокирует HTTP-запросы.

Раз в воскресенье он:

1. Берёт пользователей с Telegram ID.
2. Собирает статистику последних семи дней из SQLite.
3. Формирует prompt с учебой, сном и тренировками.
4. Отправляет prompt в Gemini.
5. Отправляет короткий отчёт пользователю через Telegram.

Конфигурация:

```env
GEMINI_API_KEY=...
GEMINI_MODEL=gemini-3.6-flash
```

Модель вынесена в env, чтобы её можно было заменить без изменения кода.

Если Google возвращает ошибку о недоступной модели, нужно запросить список моделей и выбрать модель, поддерживающую `generateContent`.

## 7. Как запустить локально

### Frontend preview

```powershell
cd C:\Users\nurta\Documents\проект\pixellife-tracker\frontend
npm.cmd install
npm.cmd run dev
```

Frontend будет доступен на:

```text
http://localhost:5173
```

### Backend на Windows

Backend использует `go-sqlite3`, которому нужен CGO и C-компилятор.

Нужны:

- Go;
- GCC/MinGW или другой C compiler;
- `CGO_ENABLED=1`.

После установки компилятора:

```powershell
cd C:\Users\nurta\Documents\проект\pixellife-tracker\backend
$env:CGO_ENABLED="1"
go test ./...
go run .\cmd\server
```

Без GCC backend на Windows завершится с ошибкой:

```text
go-sqlite3 requires cgo to work
```

Это ограничение локального Windows-окружения, а не Docker/VPS-сборки.

## 8. Как поднять на VPS через Docker

На Ubuntu VPS должны быть установлены Docker Engine и Docker Compose plugin.

В корне проекта:

```bash
cp .env.example .env
nano .env
```

Для приватного single-user режима:

```env
APP_ENV=production
PERSONAL_AUTH=true
PORT=8080
FRONTEND_URL=http://IP_СЕРВЕРА:8080
DB_PATH=/data/pixellife.db
JWT_SECRET=длинный-случайный-секрет
TELEGRAM_BOT_TOKEN=новый-токен-от-BotFather
GEMINI_API_KEY=новый-ключ-Google-AI-Studio
GEMINI_MODEL=gemini-3.6-flash
```

Запуск:

```bash
docker compose up -d --build
```

Проверка контейнера:

```bash
docker compose ps
docker compose logs --tail=200 app
curl http://127.0.0.1:8080/api/health
```

Открыть приложение:

```text
http://IP_СЕРВЕРА:8080
```

Данные SQLite лежат в Docker volume `pixellife_data`, поэтому пересборка контейнера не удаляет базу.

Обновление:

```bash
git pull
docker compose up -d --build
docker image prune -f
```

## 9. Почему раньше появлялось «backend недоступен»

Причин было несколько:

1. Backend не был запущен на Windows.
2. На Windows отсутствовал GCC для `go-sqlite3`.
3. Старый JWT в браузере мог быть недействительным.
4. Compose принудительно ставил `APP_ENV=production`, а frontend отправлял тестовые Telegram данные.
5. Некоторые формы скрывали реальную ошибку и говорили, будто данные сохранены локально.

Сейчас:

- Compose использует `.env`;
- есть `PERSONAL_AUTH=true` для приватного режима;
- frontend обновляет JWT после 401;
- есть `/api/health`;
- интерфейс не утверждает, что запись сохранена, если backend её не принял;
- Dockerfile сам устанавливает GCC и собирает CGO SQLite в Linux-контейнере.

## 10. Безопасность ключей

`.env` добавлен в `.gitignore`.

Нельзя:

- коммитить `.env`;
- вставлять токены в исходники;
- использовать токены, опубликованные в чате или логах;
- оставлять `PERSONAL_AUTH=true`, если приложение открыто для всех.

Telegram token и Gemini API key, которые были отправлены в чат, нужно перевыпустить перед реальным VPS-деплоем.

## 11. Проверки проекта

Frontend:

```bash
cd frontend
npm run build
```

Backend:

```bash
cd backend
go test ./...
```

Текущая реализация проходила обе проверки.
