# Simple VPN Bot

Telegram‑бот, который выдаёт VPN‑ключи (vless/vmess) только одному разрешённому пользователю. Ключи задаются через переменные окружения.

## Локальный запуск

1. Скопируй конфиг и заполни значения:
   ```bash
   cp .env.example .env
   # Отредактируй .env: TELEGRAM_BOT_TOKEN, ALLOWED_USER_ID, VPN_KEY_*
   ```

2. Запуск:
   ```bash
   go run .
   ```
   Или с автоперезапуском: `make run`

## Деплой на прод

### Вариант 1: VPS (systemd)

1. Клонируй репозиторий на сервер:
   ```bash
   git clone https://github.com/YOUR_USER/simple_vpn_bot.git
   cd simple_vpn_bot
   ```

2. Создай `.env` на сервере (не копируй с локальной машины по незащищённому каналу — лучше ввести вручную или через `scp`):
   ```bash
   cp .env.example .env
   nano .env
   ```

3. Собери бинарник:
   ```bash
   go build -o simple_vpn_bot .
   ```

4. Создай unit для systemd `/etc/systemd/system/simple-vpn-bot.service`:
   ```ini
   [Unit]
   Description=Simple VPN Telegram Bot
   After=network.target

   [Service]
   Type=simple
   User=YOUR_USER
   WorkingDirectory=/path/to/simple_vpn_bot
   ExecStart=/path/to/simple_vpn_bot/simple_vpn_bot
   Restart=always
   RestartSec=5
   EnvironmentFile=/path/to/simple_vpn_bot/.env

   [Install]
   WantedBy=multi-user.target
   ```

5. Запуск и автозапуск:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable simple-vpn-bot
   sudo systemctl start simple-vpn-bot
   sudo systemctl status simple-vpn-bot
   ```

### Вариант 2: Docker

В корне можно добавить `Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /simple_vpn_bot .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /simple_vpn_bot .
CMD ["./simple_vpn_bot"]
```

Сборка и запуск (переменные из `.env` или задать в `docker run -e ...`):

```bash
docker build -t simple-vpn-bot .
docker run -d --restart=unless-stopped --env-file .env simple-vpn-bot
```

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `TELEGRAM_BOT_TOKEN` | Токен от @BotFather |
| `ALLOWED_USER_ID` | Telegram user_id единственного пользователя (число) |
| `VPN_KEY_RUSSIA` | Строка ключа для России (vless://... или vmess://...) |
| `VPN_KEY_NETHERLANDS` | Строка ключа для Нидерландов |

Файл `.env` в репозиторий не коммитится — на проде создаётся вручную из `.env.example`.
