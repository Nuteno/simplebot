build:
	go build -o simple_vpn_bot .

deploy: build
	./simple_vpn_bot

docker-build:
	docker compose build bot

docker-up:
	docker compose up -d --build bot

docker-down:
	docker compose down

docker-restart:
	docker compose restart bot

docker-logs:
	docker compose logs -f --tail=200 bot

docker-status:
	docker compose ps

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
