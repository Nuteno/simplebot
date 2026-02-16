# Сервер для деплоя: user@host (задай в окружении или при вызове make deploy SERVER=user@host)
SERVER ?=
# Путь к проекту на сервере (должен совпадать с тем, откуда запускается systemd)
DEPLOY_PATH ?= ~/simple_vpn_bot

run:
	@echo "Запуск простого VPN‑бота (будет автоматически перезапускаться при падении)..."
	@while true; do \
		go run main.go || echo "Бот завершился с ошибкой, перезапуск через 1 секунду..."; \
		sleep 1; \
	done

# Деплой на сервер: git pull, go build, перезапуск systemd-сервиса simple-vpn-bot.
# Один раз на сервере: клонируй репо, настрой .env, создай и включи simple-vpn-bot.service.
deploy:
	@test -n "$(SERVER)" || (echo "Укажи сервер: make deploy SERVER=user@host"; exit 1)
	ssh $(SERVER) "cd $(DEPLOY_PATH) && git pull && go build -o simple_vpn_bot . && sudo systemctl restart simple-vpn-bot && sudo systemctl status simple-vpn-bot --no-pager"

