package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Чувствительные данные читаем из переменных окружения / .env.
var (
	// TELEGRAM_BOT_TOKEN — токен бота от @BotFather.
	botToken string

	// ALLOWED_USER_ID — единственный пользователь, который может пользоваться ботом.
	// Узнать свой ID можно, например, через @userinfobot или любой другой сервис.
	allowedUserID int64

	// VPN_KEY_RUSSIA и VPN_KEY_NETHERLANDS — VPN‑ключи для разных стран.
	vpnKeyRussia      string
	vpnKeyNetherlands string
)

// Тексты сообщений
const (
	textInstruction = "📖 Инструкция по использованию VPN:\n\n" +
		"1. Нажмите кнопку «Получить VPN».\n" +
		"2. Выберите нужную страну: Россия или Нидерланды.\n" +
		"3. Скопируйте выданный ключ и импортируйте его в своё VPN‑приложение.\n\n" +
		"Ключи зашиты напрямую в код бота и не зависят от базы данных или внешних сервисов."

	textStart = "👋 Привет! Этот бот выдаёт VPN‑ключ только одному авторизованному пользователю.\n\n" +
		"Используйте кнопки ниже, чтобы получить инструкцию или VPN‑ключ."

	textAccessDenied = "⛔ У вас нет доступа к этому боту."
)

// Тексты кнопок
const (
	btnInstruction = "📖 Инструкция"
	btnGetVPN      = "Получить VPN"
)

func main() {
	// Пытаемся прочитать .env из текущей директории (если файла нет — просто продолжаем).
	loadEnvFile(".env")

	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	allowedUserIDStr := os.Getenv("ALLOWED_USER_ID")
	vpnKeyRussia = os.Getenv("VPN_KEY_RUSSIA")
	vpnKeyNetherlands = os.Getenv("VPN_KEY_NETHERLANDS")

	if botToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_BOT_TOKEN не задана")
	}

	if allowedUserIDStr == "" {
		log.Fatal("Переменная окружения ALLOWED_USER_ID не задана")
	}

	parsedUserID, err := strconv.ParseInt(allowedUserIDStr, 10, 64)
	if err != nil {
		log.Fatalf("ALLOWED_USER_ID должна быть числом (int64): %v", err)
	}
	allowedUserID = parsedUserID

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Debug = false
	log.Printf("Бот запущен как @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// Общая проверка доступа по user_id
		if !isAllowed(update) {
			denyAccess(bot, update)
			continue
		}

		if update.Message != nil {
			handleMessage(bot, update.Message)
		}

		if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery)
		}
	}
}

// Простейший загрузчик .env‑файла формата KEY=VALUE.
// Строки, начинающиеся с #, и пустые строки игнорируются.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// .env отсутствует — это не ошибка: переменные могут быть заданы снаружи.
			return
		}
		log.Fatalf("Ошибка открытия .env файла %s: %v", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Уберём обрамляющие кавычки, если есть.
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') ) {
			value = value[1 : len(value)-1]
		}

		if err := os.Setenv(key, value); err != nil {
			log.Printf("Не удалось установить переменную окружения %s: %v", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Ошибка чтения .env файла %s: %v", path, err)
	}
}

// Проверка, что апдейт пришёл от разрешённого пользователя.
func isAllowed(update tgbotapi.Update) bool {
	switch {
	case update.Message != nil && update.Message.From != nil:
		return update.Message.From.ID == allowedUserID
	case update.CallbackQuery != nil && update.CallbackQuery.From != nil:
		return update.CallbackQuery.From.ID == allowedUserID
	default:
		return false
	}
}

// Отправляем отказ в доступе всем, кроме разрешённого пользователя.
func denyAccess(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	var chatID int64

	switch {
	case update.Message != nil:
		chatID = update.Message.Chat.ID
	case update.CallbackQuery != nil && update.CallbackQuery.Message != nil:
		chatID = update.CallbackQuery.Message.Chat.ID
	default:
		return
	}

	// Можно молча игнорировать, но для наглядности отправим сообщение.
	msg := tgbotapi.NewMessage(chatID, textAccessDenied)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения об отказе в доступе: %v", err)
	}
}

// Главная клавиатура с двумя кнопками.
func mainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(btnInstruction),
			tgbotapi.NewKeyboardButton(btnGetVPN),
		),
	)
}

// Обработка обычных сообщений.
func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	switch {
	case msg.IsCommand() && msg.Command() == "start":
		sendStart(bot, msg.Chat.ID)
	case msg.Text == btnInstruction || msg.Text == "Инструкция":
		sendInstruction(bot, msg.Chat.ID)
	case msg.Text == btnGetVPN:
		sendCountrySelection(bot, msg.Chat.ID)
	default:
		// Можно ничего не отвечать или подсказать доступные действия.
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Выберите действие с помощью кнопок ниже.")
		reply.ReplyMarkup = mainKeyboard()
		if _, err := bot.Send(reply); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}

// Обработка нажатий на inline‑кнопки.
func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	switch cb.Data {
	case "country_russia":
		sendVPNKey(bot, cb.Message.Chat.ID, "🇷🇺 Россия", vpnKeyRussia)
	case "country_netherlands":
		sendVPNKey(bot, cb.Message.Chat.ID, "🇳🇱 Нидерланды", vpnKeyNetherlands)
	default:
		// Неизвестный callback — игнорируем.
	}

	// Уберём "часики" у кнопки.
	if _, err := bot.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
		log.Printf("Ошибка отправки callback‑ответа: %v", err)
	}
}

func sendStart(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, textStart)
	msg.ReplyMarkup = mainKeyboard()
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки стартового сообщения: %v", err)
	}
}

func sendInstruction(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, textInstruction)
	msg.ReplyMarkup = mainKeyboard()
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки инструкции: %v", err)
	}
}

// Отправляем выбор страны через inline‑кнопки.
func sendCountrySelection(bot *tgbotapi.BotAPI, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Россия", "country_russia"),
			tgbotapi.NewInlineKeyboardButtonData("🇳🇱 Нидерланды", "country_netherlands"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Выберите страну для получения VPN‑ключа:")
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки выбора страны: %v", err)
	}
}

// Отправка конкретного VPN‑ключа.
func sendVPNKey(bot *tgbotapi.BotAPI, chatID int64, countryLabel, key string) {
	if key == "" {
		msg := tgbotapi.NewMessage(chatID, "Ключ для "+countryLabel+" ещё не настроен. Обратитесь к администратору.")
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения о ненастроенном ключе: %v", err)
		}
		return
	}

	// Показываем ключ как обычный текст, но в виде code-блока,
	// чтобы Telegram не превращал его в кликабельную ссылку.
	escapedKey := strings.ReplaceAll(key, "&", "&amp;")
	escapedKey = strings.ReplaceAll(escapedKey, "<", "&lt;")
	escapedKey = strings.ReplaceAll(escapedKey, ">", "&gt;")

	text := "Ваш VPN‑ключ для " + countryLabel + ":\n\n" +
		"<code>" + escapedKey + "</code>"

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки VPN‑ключа: %v", err)
	}

	// Дополнительно можно отправить напоминание или лог.
	log.Printf("Выдан VPN‑ключ для %s пользователю %d в %s", countryLabel, allowedUserID, time.Now().Format(time.RFC3339))
}

