package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"time"
	"fmt"
)

var messageChan = make(chan *tgbotapi.MessageConfig, 5)
var callbackChan = make(chan tgbotapi.CallbackConfig, 5)

func notifyUser(id int64, message string) {
	m := tgbotapi.NewMessage(id, message)
	messageChan <- &m
}

func monitor() {
	for _ = range time.Tick(30*time.Second) {
		ul := userList()
		clearTableCache()
		for _, u := range ul {
			pairs := recordList(u)
			for _, v := range pairs {
				data := parseList(v.Value)
				cellval, err := cellValueByRecord(data)
				if err != nil {
					println(err.Error())
					continue
				}
				if cellval == nil {
					println("Could not fetch value");
					continue;
				}
				old := updateCellVal(u, v.Name, *cellval)
				if old != *cellval {
					notifyUser(u, fmt.Sprintf("The cell %s has changed!\n'%s' -> '%s'", v.Name, old, *cellval))
				}
			}
		}
	}
}

func sender(bot *tgbotapi.BotAPI) {
	for ;; {
		select {
		case m := <-messageChan:
			if m != nil {
				bot.Send(m)
			}
		case m := <-callbackChan:
			bot.AnswerCallbackQuery(m)
		}
	}
}

func main() {
	connect()
	bot, err := tgbotapi.NewBotAPI(configMap["token"])
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	go monitor()
	go sender(bot)

	for update := range updates {
		if update.CallbackQuery != nil {
			log.Printf("[%s CALLBACK] %s", update.CallbackQuery.From.UserName, update.CallbackQuery.Data)
			messageChan <- handleCallback(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
			callbackChan <- tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			continue
		}
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		messageChan <- handle(update.Message.Chat.ID, update.Message.Text)
	}
}
