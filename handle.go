package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
	"encoding/json"
	"strconv"
)

var state = make(map[int64]map[string]string)
var MENU_KB = []string{"Add a cell", "List all cells", "Delete a cell"}

const HELP_STR = `You can add cells here. I will check them about once a minute, and if the cell value changes, I will notify you.
`

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

func makeMessage(id int64, text string, kb []string) *tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = makeKeyboard(kb)
	msg.DisableWebPagePreview = true
	return &msg
}

func formatRecordList(uid int64, names []string, values []string) string {
	if len(names) == 0 {
		return "You have no cells yet"
	}
	res := "Your cells:\n\n"
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

func sendInitialValue(uid int64, record []string) {
	var val string
	cellval, err := extractCellValue(getTable(record[0]), record[1], record[3], record[2])
	if err == nil {
		val = "\nInitial value: '" + cellval + "'"
	}
	messageChan <- makeMessage(uid, "New cell added!"+val, MENU_KB)
}

func handle(id int64, message string) *tgbotapi.MessageConfig {
	ustate, ok := state[id]
	if !ok {
		state[id] = make(map[string]string)
		ustate = state[id]
	}
	if message == "/start" {
		ustate["name"] = ""
		return makeMessage(id, "Hello!", MENU_KB)
	}
	if message == "/help" {
		ustate["name"] = ""
		return makeMessage(id, HELP_STR, MENU_KB)
	}
	if message == "Cancel" {
		ustate["name"] = ""
		return makeMessage(id, "Ok", MENU_KB)
	}
	switch ustate["name"] {
	case "":
		switch message {
		case MENU_KB[0]:
			ustate["name"] = "add"
			return makeMessage(id, "Enter the cell URL. You can get it by right-clicking the cell and copying the link to it.", []string{"Cancel"})
		case MENU_KB[1]:
			names, values := recordList(id)
			return makeMessage(id, formatRecordList(id, names, values), MENU_KB)
		case MENU_KB[2]:
			names, values := recordList(id)
			if len(names) == 0 {
				return makeMessage(id, "You have no cells yet", MENU_KB)
			}
			ustate["name"] = "delete"
			data, _ := json.Marshal(names)
			ustate["record-names"] = string(data)
			return makeMessage(id, formatRecordList(id, names, values)+"\nWhich cells do you want to delete?\n" +
				"Enter their numbers separated by commas or spaces", []string{"Cancel"})
		default:
			return makeMessage(id, "Wat?", MENU_KB)
		}
	case "add":
		message = strings.Trim(message, " ")
		parsed := parseURL(message)
		if len(parsed) != 4 {
			return makeMessage(id, "Invalid url, try again.", []string{"Cancel"})
		}
		data, err := json.Marshal(parsed)
		if err != nil {
			return makeMessage(id, "Something went wrong", []string{"Cancel"})
		}
		ustate["record"] = string(data)
		ustate["name"] = "add-name"
		return makeMessage(id, "Enter the name for this cell", []string{"Cancel"})
	case "add-name":
		message = strings.Trim(message, " ")
		if len(message) == 0 {
			return makeMessage(id, "Bad name, try again", []string{"Cancel"})
		}
		if recordExists(id, message) {
			return makeMessage(id, "This name is already used, try again", []string{"Cancel"})
		}
		deleteCellVal(id, message)
		go sendInitialValue(id, parseList(ustate["record"]))
		addRecord(id, message, ustate["record"])
		ustate["name"] = ""
		return nil
	case "delete":
		message = strings.Trim(message, " ")
		numbers := strings.Split(strings.Replace(message, ",", " ", -1), " ")
		ints := make([]int64, 0)
		names := parseList(ustate["record-names"])
		for _, val := range numbers {
			if val == "" {
				continue
			}
			res, err := strconv.ParseInt(val, 10, 64)
			if err != nil || res <= 0 || res > int64(len(names)) {
				return makeMessage(id, "Bad number, try again", []string{"Cancel"})
			}
			ints = append(ints, res)
		}
		for _, num := range ints {
			deleteRecord(id, names[num-1])
		}
		ustate["name"] = ""
		return makeMessage(id, "Deleted!", MENU_KB)
	}
	return makeMessage(id, "Not implemented yet", MENU_KB)
}
