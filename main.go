package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/KMR3X/ISaverUB/config"
	database "github.com/KMR3X/ISaverUB/internal"

	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"
)

// Запуск и подключение к бд
var session = database.ConnectDB()

// Инициализация переменных клиента и контекста
var client *gotgproto.Client
var clientCtx *ext.Context

func main() {
	//Завершение сессии по окончании выполнения main
	defer session.Close()

	//Инициализация ConfigInfo
	config.Init()
	appid, _ := strconv.Atoi(config.Get().Auth.AppId)

	clientType := gotgproto.ClientType{
		Phone: config.Get().Auth.PhoneNum,
	}
	//инициализация клиента и вход в тг
	client, _ = gotgproto.NewClient(
		appid,
		config.Get().Auth.AppHash,
		clientType,
		&gotgproto.ClientOpts{
			Session:          sessionMaker.SqliteSession("saver7xbot"),
			DisableCopyright: true,
		},
	)

	clientCtx = client.CreateContext()

	dispatcher := client.Dispatcher
	//Обработчик, вызывающий сохранение информации для любого сообщения
	dispatcher.AddHandler(handlers.NewAnyUpdate(getAndSaveUserInfo))
	//Обработчик, проверяющий наличие ссылки в сообщении
	dispatcher.AddHandler(handlers.NewMessage(filters.Message.Text, linkCheck))

	fmt.Printf("Запуск клиента @%s...\n", client.Self.FirstName)
	client.Idle()
}

// получение информации о пользователе и передача на запись
func getAndSaveUserInfo(ctx *ext.Context, update *ext.Update) error {
	if update.EffectiveMessage == nil || update.EffectiveUser() == nil {
		return nil
	}

	//Запись информации о пользователе в переменную user
	var msgText string
	var user = database.Record{
		ID:           strconv.Itoa(int(update.EffectiveUser().ID)),
		IsBot:        strconv.FormatBool(update.EffectiveUser().Bot),
		FirstName:    update.EffectiveUser().FirstName,
		LastName:     update.EffectiveUser().LastName,
		UserName:     update.EffectiveUser().Username,
		LanguageCode: update.EffectiveUser().LangCode,
	}

	//Проверка на наличие пользователя в бд, если нет - сохранение его данных
	if database.SelectQuery(session, user.ID) {
		msgText = "Пользователь уже записан."
		fmt.Println("Отказано в добавлении: пользователь уже существует.")
	} else {
		database.InsertQuery(session, user)
		msgText = "Пользователь добавлен."
		fmt.Println("Успешно: пользователь добавлен.")
	}

	//Сборка сообщения
	msg := tg.MessagesSendMessageRequest{
		Message: msgText,
	}
	//Отправка сообщения со статусом
	_, err := ctx.SendMessage(update.EffectiveChat().GetID(), &msg)
	return err
}

var joinLink string

// Проверка ссылки на валидность, вступление в группу/канал
func linkCheck(ctx *ext.Context, update *ext.Update) error {
	var valid bool = false
	var linkPrefix, linkPostfix bool = false, false
	var prefixes = []string{
		"t.me/",
		"http://t.me/",
		"https://t.me/",
		"telegram.me/",
		"http://telegram.me/",
		"https://telegram.me/",
		"telegram.dog/",
		"http://telegram.dog/",
		"https://telegram.dog/",
		"tg:",
		"tg://",
	}
	var disallowedSubdomains = []string{
		"addemoji", "addlist", "addstickers", "addtheme", "auth", "boost", "confirmphone",
		"contact", "giftcode", "invoice", "joinchat", "login", "proxy", "setlanguage",
		"share", "socks", "web", "a", "k", "z", "www",
	}

	//Проверка на наличие в сообщении префикса ссылки
	for _, prefix := range prefixes {
		if strings.HasPrefix(update.EffectiveMessage.Text, prefix) {
			valid = true
			linkPrefix = true
		}
	}
	//Проверка на наличие в сообщении постфикса ссылки и доп.условий валидности
	if strings.HasSuffix(update.EffectiveMessage.Text, ".t.me") {
		//часть ссылки без постфикса .t.me
		linkWOPF := strings.TrimSuffix(update.EffectiveMessage.Text, ".t.me")
		valid = true
		linkPostfix = true

		//проход по всем подстрокам из массива запрещенных
		for _, ds := range disallowedSubdomains {
			//если подстрока включена,
			if strings.HasPrefix(update.EffectiveMessage.Text, ds) {
				//проверка на полное соответствие подстроки и ссылки
				if linkWOPF == update.EffectiveMessage.Text {
					valid = false
					linkPostfix = false
					break
				}
				//Если не включена,
				//Проверка на единичную длину ссылки
			} else if len(linkWOPF) <= 1 {
				valid = false
				linkPostfix = false
				break
			}
		}
	}

	//Если у ссылки есть и префикс, и постфикс, то она не валидна
	if linkPrefix && linkPostfix {
		valid = false
	}

	if valid {
		//fmt.Println("+++валидная ссылка+++")
		joinLink = update.EffectiveMessage.Text
		err := JoinGroupChannel(clientCtx, update)
		if err != nil {
			log.Fatal("Не удалось присоединиться: ", err)
		}
	} //else {
	//fmt.Println("---невалидная ссылка---")
	//}
	return nil
}

// Вход в канал или группу
func JoinGroupChannel(ctx *ext.Context, update *ext.Update) error {
	var msg tg.MessagesSendMessageRequest

	sender := message.NewSender(tg.NewClient(client))

	//Если ссылка имеет join или +, то:
	if strings.Contains(joinLink, "+") || strings.Contains(joinLink, "join") {
		fmt.Printf("\nПрисоединение к чату по приглашению %s\n", joinLink)

		_, err := sender.JoinLink(ctx, joinLink)
		if err != nil {
			fmt.Println("Ошибка присоединения: ", err)
			return err
		} else {
			//Сборка сообщения
			msg = tg.MessagesSendMessageRequest{
				Message: "Чат добавлен.",
			}
		}
	} else {
		//Если обычная ссылка на канал без присоединения, то:
		fmt.Printf("\nПрисоединение к каналу по ссылке %s\n", joinLink)
		_, err := sender.Resolve(joinLink).Join(clientCtx)

		if err != nil {
			fmt.Println("Ошибка присоединения: ", err)
			return err
		} else {
			//Сборка сообщения
			msg = tg.MessagesSendMessageRequest{
				Message: "Группа добавлена.",
			}
		}
	}
	//Отправка сообщения со статусом
	_, err := ctx.SendMessage(update.EffectiveChat().GetID(), &msg)
	return err
}
