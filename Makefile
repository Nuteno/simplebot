build:
	go build -o simple_vpn_bot .

deploy: build
	./simple_vpn_bot

run:
	@echo "Запуск с автоперезапуском (go run)..."
	@while true; do \
		go run . || (echo "Перезапуск через 1 сек..."; sleep 1); \
	done
