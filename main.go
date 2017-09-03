package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"time"
	"fmt"
)

var messageChan = make(chan tgbotapi.MessageConfig, 5)

func notifyUser(id int64, message string) {
	messageChan <- tgbotapi.NewMessage(id, message)
}

func monitor() {
	for _ = range time.Tick(30*time.Second) {
		ul := userList()
		clearTableCache()
		for _, u := range ul {
			names, values := recordList(u)
			for i, v := range values {
				data := parseList(v)
				cellval, err := extractCellValue(getTable(data[0]), data[1], data[3], data[2])
				if err != nil {
					println(err.Error())
					continue
				}
				old := updateCellVal(u, names[i], cellval)
				println(names[i], cellval)
				if old != cellval {
					notifyUser(u, fmt.Sprintf("The cell %s has changed!\n'%s' -> '%s'", names[i], old, cellval))
				}
			}
		}
	}
}

func sender(bot *tgbotapi.BotAPI) {
	for m := range messageChan {
		bot.Send(m)
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
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		messageChan <- handle(update.Message.Chat.ID, update.Message.Text)
	}
}
