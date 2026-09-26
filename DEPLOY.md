# PixelLife VPS deploy

## 1. Prepare the server

Use an Ubuntu VPS with Docker Engine and Compose plugin installed. Open TCP port 8080 in the VPS firewall, or put the app behind Nginx/Caddy.

```bash
git clone <your-repository-url> pixellife-tracker
cd pixellife-tracker
cp .env.example .env
nano .env
```

Set these values in `.env`:

```env
APP_ENV=production
PERSONAL_AUTH=true
FRONTEND_URL=http://YOUR_DOMAIN_OR_IP:8080
JWT_SECRET=generate-a-long-random-secret
TELEGRAM_BOT_TOKEN=the-token-from-botfather
GEMINI_API_KEY=the-key-from-google-ai-studio
GEMINI_MODEL=gemini-3.6-flash
```

Never commit `.env`. Rotate any key that was shared in chat or logs.

## 2. Build and run

```bash
docker compose up -d --build
docker compose logs -f app
```

The app is available at `http://YOUR_DOMAIN_OR_IP:8080`.

Check the backend before opening the UI:

```bash
curl http://127.0.0.1:8080/api/health
```

The response must contain `"status":"ok"`. `PERSONAL_AUTH=true` is intended for this private, single-user tracker: it lets the browser create the local personal account. For a public deployment, set it to `false` and implement Telegram WebApp `init_data` login before exposing the app.

## 3. Verify integrations

The backend log must contain either:

- `Telegram bot connected`
- `Telegram disabled: ...` with the exact API error

If the UI shows `backend недоступен`, run:

```bash
docker compose ps
docker compose logs --tail=200 app
curl http://127.0.0.1:8080/api/health
```

If the health endpoint is OK but writes return `401`, clear the browser site data and reload. The app will issue a fresh personal JWT.

The Telegram bot uses long polling, so no public webhook is required. Send `/start`, `/sleep 7.5`, `/sport`, or `/shelf Dune` to the bot.

Gemini reports are generated asynchronously on Sundays. The database must contain at least one Telegram-linked user before a report can be sent.

## 4. Update the deployment

```bash
git pull
docker compose up -d --build
docker image prune -f
```

SQLite data is stored in the `pixellife_data` Docker volume and survives container rebuilds.
