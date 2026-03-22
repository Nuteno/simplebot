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

// parseAllowedUserIDs читает ALLOWED_USER_IDS (через запятую) или одиночный ALLOWED_USER_ID.
func parseAllowedUserIDs() (map[int64]struct{}, error) {
	out := make(map[int64]struct{})
	raw := strings.TrimSpace(os.Getenv("ALLOWED_USER_IDS"))
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, err
			}
			out[id] = struct{}{}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	}
	single := strings.TrimSpace(os.Getenv("ALLOWED_USER_ID"))
	if single == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(single, 10, 64)
	if err != nil {
		return nil, err
	}
	out[id] = struct{}{}
	return out, nil
}

// Чувствительные данные читаем из переменных окружения / .env.
var (
	// TELEGRAM_BOT_TOKEN — токен бота от @BotFather.
	botToken string

	// allowedUserIDs — пользователи, которым разрешён доступ к боту (ALLOWED_USER_IDS или ALLOWED_USER_ID).
	allowedUserIDs map[int64]struct{}

	// VPN_KEY_* — VPN‑ключи для разных стран.
	vpnKeyRussia      string
	vpnKeyNetherlands string
	vpnKeyUAE         string
	vpnKeyTurkey      string
	vpnKeySingapore   string
	vpnKeyKazakhstan  string
)

// Тексты сообщений
const (
	textInstruction = "📖 Инструкция по использованию VPN:\n\n" +
		"1. Нажмите кнопку «Получить VPN».\n" +
		"2. Выберите нужную страну в списке.\n" +
		"3. Скопируйте выданный ключ и импортируйте его в своё VPN‑приложение.\n\n" +
		"Ключи зашиты напрямую в код бота и не зависят от базы данных или внешних сервисов."

	textStart = "👋 Привет! Этот бот выдаёт VPN‑ключ только авторизованным пользователю.\n\n" +
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
	vpnKeyRussia = os.Getenv("VPN_KEY_RUSSIA")
	vpnKeyNetherlands = os.Getenv("VPN_KEY_NETHERLANDS")
	vpnKeyUAE = os.Getenv("VPN_KEY_UAE")
	vpnKeyTurkey = os.Getenv("VPN_KEY_TURKEY")
	vpnKeySingapore = os.Getenv("VPN_KEY_SINGAPORE")
	vpnKeyKazakhstan = os.Getenv("VPN_KEY_KAZAKHSTAN")

	if botToken == "" {
		log.Fatal("Переменная окружения TELEGRAM_BOT_TOKEN не задана")
	}

	var err error
	allowedUserIDs, err = parseAllowedUserIDs()
	if err != nil {
		log.Fatalf("Некорректный список разрешённых пользователей: %v", err)
	}
	if len(allowedUserIDs) == 0 {
		log.Fatal("Задайте ALLOWED_USER_IDS (через запятую) или ALLOWED_USER_ID")
	}

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
	var id int64
	switch {
	case update.Message != nil && update.Message.From != nil:
		id = update.Message.From.ID
	case update.CallbackQuery != nil && update.CallbackQuery.From != nil:
		id = update.CallbackQuery.From.ID
	default:
		return false
	}
	_, ok := allowedUserIDs[id]
	return ok
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
		sendVPNKey(bot, cb.Message.Chat.ID, cb.From.ID, "🇷🇺 Россия", vpnKeyRussia)
	case "country_netherlands":
		sendVPNKey(bot, cb.Message.Chat.ID, cb.From.ID, "🇳🇱 Нидерланды", vpnKeyNetherlands)
	case "country_uae":
		sendVPNKey(bot, cb.Message.Chat.ID, cb.From.ID, "🇦🇪 ОАЭ", vpnKeyUAE)
	case "country_turkey":
		sendVPNKey(bot, cb.Message.Chat.ID, cb.From.ID, "🇹🇷 Турция", vpnKeyTurkey)
	case "country_singapore":
		sendVPNKey(bot, cb.Message.Chat.ID, cb.From.ID, "🇸🇬 Сингапур", vpnKeySingapore)
	case "country_kazakhstan":
		sendVPNKey(bot, cb.Message.Chat.ID, cb.From.ID, "🇰🇿 Казахстан", vpnKeyKazakhstan)
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
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇦🇪 ОАЭ", "country_uae"),
			tgbotapi.NewInlineKeyboardButtonData("🇹🇷 Турция", "country_turkey"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇸🇬 Сингапур", "country_singapore"),
			tgbotapi.NewInlineKeyboardButtonData("🇰🇿 Казахстан", "country_kazakhstan"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "Выберите страну для получения VPN‑ключа:")
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки выбора страны: %v", err)
	}
}

// Отправка конкретного VPN‑ключа.
func sendVPNKey(bot *tgbotapi.BotAPI, chatID, fromUserID int64, countryLabel, key string) {
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
	log.Printf("Выдан VPN‑ключ для %s пользователю %d в %s", countryLabel, fromUserID, time.Now().Format(time.RFC3339))
}

