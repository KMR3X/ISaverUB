package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/KMR3X/ISaverUB/config"
	database "github.com/KMR3X/ISaverUB/internal"
	"github.com/gotd/td/tg"

	gotgproto "github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/ext"
	sessionMaker "github.com/celestix/gotgproto/sessionMaker"
)

// Запуск и подключение к бд
var session = database.ConnectDB()

func main() {

	defer session.Close()

	//ConfigInfo
	config.Init()
	appid, _ := strconv.Atoi(config.Get().Auth.AppId)

	clientType := gotgproto.ClientType{
		Phone: config.Get().Auth.PhoneNum,
	}

	//инициализация клиента и вход в тг
	client, err := gotgproto.NewClient(
		//AppID
		appid,
		//ApiHash
		config.Get().Auth.AppHash,
		//Тип клиента
		clientType,
		//Опциональные параметры клиента
		&gotgproto.ClientOpts{
			Session: sessionMaker.SqliteSession("saver7xbot"),
		},
	)
	if err != nil {
		log.Fatalln("Ошибка запуска:", err)
	}

	dispatcher := client.Dispatcher

	//Обработчик, вызывающий сохранятор информации для любого сообщения
	dispatcher.AddHandler(handlers.NewAnyUpdate(getAndSaveUserInfo))

	fmt.Printf("Запуск клиента @%s...\n", client.Self.FirstName)

	client.Idle()
}

// получение информации о пользователе и передача на запись
func getAndSaveUserInfo(ctx *ext.Context, update *ext.Update) error {
	if update.EffectiveMessage == nil || update.EffectiveUser() == nil {
		return nil
	}

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
		msgText = "Такой пользователь уже существует. "
		fmt.Println("Отказано в добавлении: пользователь уже существует.")
	} else {
		database.InsertQuery(session, user)
		msgText = "Информация сохранена. "
		fmt.Println("Успешно: пользователь добавлен.")
	}

	msg := tg.MessagesSendMessageRequest{
		Message: msgText,
	}
	_, err := ctx.SendMessage(update.EffectiveChat().GetID(), &msg)
	return err
}
