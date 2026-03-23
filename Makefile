DOCKER := $(shell command -v docker 2>/dev/null)
DOCKER_COMPOSE_BIN := $(shell command -v docker-compose 2>/dev/null)
DOCKER_COMPOSE_PLUGIN_OK := $(shell docker compose version >/dev/null 2>&1 && echo yes)

ifeq ($(DOCKER),)
COMPOSE_CMD := @echo "Docker не установлен. Установите docker.io (и docker-compose-plugin)"; false
else ifneq ($(DOCKER_COMPOSE_BIN),)
COMPOSE_CMD := docker-compose
else ifeq ($(DOCKER_COMPOSE_PLUGIN_OK),yes)
COMPOSE_CMD := docker compose
else
COMPOSE_CMD := @echo "Docker Compose не найден. Установите docker-compose-plugin или docker-compose"; false
endif

build:
	go build -o simple_vpn_bot .

deploy: build
	./simple_vpn_bot

docker-build:
	$(COMPOSE_CMD) build bot

docker-up:
	$(COMPOSE_CMD) up -d --build bot

docker-down:
	$(COMPOSE_CMD) down

docker-restart:
	$(COMPOSE_CMD) restart bot

docker-logs:
	$(COMPOSE_CMD) logs -f --tail=200 bot

docker-status:
	$(COMPOSE_CMD) ps

# Запуск в фоне: не умрёт при выходе из консоли. Логи — bot.log
start: build
	@nohup ./simple_vpn_bot >> bot.log 2>&1 & echo $$! > bot.pid && echo "Бот запущен в фоне (PID $$(cat bot.pid)). Логи: tail -f bot.log"

# Остановить бота (запущенного через make start)
stop:
	@if [ -f bot.pid ]; then kill $$(cat bot.pid) 2>/dev/null; rm -f bot.pid; echo "Бот остановлен"; else pkill -f simple_vpn_bot 2>/dev/null && echo "Бот остановлен" || echo "Запущенный бот не найден"; fi

run:
	@echo "Запуск с автоперезапуском (go run)..."
	@while true; do \
		go run . || (echo "Перезапуск через 1 сек..."; sleep 1); \
	done
