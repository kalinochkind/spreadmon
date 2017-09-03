package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
	"encoding/json"
	"strconv"
)

var state = make(map[int64]map[string]string)
var MENU_KB = []string{"Add", "List", "Delete"}

func makeKeyboard(kb []string) interface{} {
	if len(kb) == 0 {
		return tgbotapi.ReplyKeyboardHide{}
	}
	buttons := make([]tgbotapi.KeyboardButton, len(kb))
	for i, s := range kb {
		buttons[i] = tgbotapi.NewKeyboardButton(s)
	}
	res := tgbotapi.NewReplyKeyboard(buttons)
	res.ResizeKeyboard = false
	return res
}

func makeMessage(id int64, text string, kb []string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = makeKeyboard(kb)
	msg.DisableWebPagePreview = true
	return msg
}

func formatRecordList(uid int64, names []string, values []string) string {
	if len(names) == 0 {
		return "No records yet"
	}
	res := "Your records:\n\n"
	for i := range names {
		res += strconv.Itoa(i+1) + ". " + names[i]
		val, ok := getCellVal(uid, names[i])
		if ok {
			res += " (value: '" + val + "')"
		}
		res += "\n" + buildEditURL(parseList(values[i])) + "\n\n"
	}
	return res
}

func handle(id int64, message string) tgbotapi.MessageConfig {
	ustate, ok := state[id]
	if !ok {
		state[id] = make(map[string]string)
		ustate = state[id]
	}
	if message == "Cancel" {
		ustate["name"] = ""
		return makeMessage(id, "Ok", MENU_KB)
	}
	switch ustate["name"] {
	case "":
		switch message {
		case "Add":
			ustate["name"] = "add"
			return makeMessage(id, "Url:", []string{"Cancel"})
		case "List":
			names, values := recordList(id)
			return makeMessage(id, formatRecordList(id, names, values), MENU_KB)
		case "Delete":
			names, values := recordList(id)
			if len(names) == 0 {
				return makeMessage(id, "You have no records", MENU_KB)
			}
			ustate["name"] = "delete"
			data, _ := json.Marshal(names)
			ustate["record-names"] = string(data)
			return makeMessage(id, formatRecordList(id, names, values)+"\nWhat record do you want to delete?", []string{"Cancel"})
		default:
			return makeMessage(id, "Wat?", MENU_KB)
		}
	case "add":
		message = strings.Trim(message, " ")
		parsed := parseURL(message)
		if len(parsed) != 4 {
			return makeMessage(id, "Invalid url", []string{"Cancel"})
		}
		data, err := json.Marshal(parsed)
		if err != nil {
			return makeMessage(id, "Something went wrong", []string{"Cancel"})
		}
		ustate["record"] = string(data)
		ustate["name"] = "add-name"
		return makeMessage(id, "Name:", []string{"Cancel"})
	case "add-name":
		message = strings.Trim(message, " ")
		if len(message) == 0 {
			return makeMessage(id, "Bad name, try again", []string{"Cancel"})
		}
		if recordExists(id, message) {
			return makeMessage(id, "This name is already used, try again", []string{"Cancel"})
		}
		deleteCellVal(id, message)
		addRecord(id, message, ustate["record"])
		ustate["name"] = ""
		return makeMessage(id, "New record added!", MENU_KB)
	case "delete":
		message = strings.Trim(message, " ")
		num, err := strconv.ParseInt(message, 10, 64)
		names := parseList(ustate["record-names"])
		if err != nil || num <= 0 || num > int64(len(names)) {
			return makeMessage(id, "Bad number, try again", []string{"Cancel"})
		}
		deleteRecord(id, names[num-1])
		ustate["name"] = ""
		return makeMessage(id, "Deleted!", MENU_KB)
	}
	return makeMessage(id, "Not implemented yet", MENU_KB)
}
