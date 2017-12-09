package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
	"encoding/json"
	"strconv"
)

var state = make(map[int64]map[string]string)
var MENU_KB = []string{"Add a cell", "List all cells"}

const TABS_STR = "Monitor tabs"
const HELP_STR = `You can add cells or cell ranges here. I will check them about once a minute, and if the value changes, I will notify you.
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

func makeInlineKeyboard(kb []string) interface{} {
	buttons := make([]tgbotapi.InlineKeyboardButton, len(kb))
	for i, s := range kb {
		buttons[i] = tgbotapi.NewInlineKeyboardButtonData(s, s)
	}
	res := tgbotapi.NewInlineKeyboardMarkup(buttons)
	return res
}

func makeMessage(id int64, text string, kb []string) *tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = makeKeyboard(kb)
	msg.DisableWebPagePreview = true
	return &msg
}

func makeMessageInline(id int64, text string, kb []string) *tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(id, text)
	msg.ReplyMarkup = makeInlineKeyboard(kb)
	msg.DisableWebPagePreview = true
	return &msg
}

func formatRecordList(uid int64, pairs StringPairs) string {
	if len(pairs) == 0 {
		return "You have no cells yet"
	}
	res := "Your cells:\n\n"
	for i, v := range pairs {
		res += strconv.Itoa(i+1) + ". " + v.Name
		val, ok := getCellVal(uid, v.Name)
		if ok {
			res += " (value: '" + val + "')"
		}
		res += "\n" + buildEditURL(parseList(v.Value)) + "\n\n"
	}
	return res
}

func sendInitialValue(uid int64, record []string) {
	val := ""
	cellval, err := cellValueByRecord(record)
	if err == nil && cellval != nil {
		val = "\nInitial value: '" + *cellval + "'"
	}
	messageChan <- makeMessage(uid, "New cell added!"+val, MENU_KB)
}

func cellValueByRecord(record []string) (*string, error) {
	val := getTable(record[0])
	if val == nil {
		return nil, nil
	}
	if record[2] == "tabs" {
		return getPageListString(*val), nil
	}
	res, err := extractCellValue(*val, record[1], record[3], record[2], record[5], record[4])
	return &res, err
}

func sendPageList(uid int64, name string) {
	table := getTable(name)

	if table == nil {
		state[uid]["name"] = "add"
		messageChan <- makeMessage(uid, "Could not fetch the table, try again", []string{"Cancel"})
		return
	}
	names, gids := getPageList(*table)
	if names == nil {
		state[uid]["name"] = "add"
		messageChan <- makeMessage(uid, "Invalid table, try again", []string{"Cancel"})
		return
	}
	if len(names) == 1 {
		data := parseList(state[uid]["record"])
		data[1] = gids[0]
		cdata, _ := json.Marshal(data)
		state[uid]["record"] = string(cdata)
		state[uid]["name"] = "add-cell"
		messageChan <- makeMessage(uid, "What cell do you want to monitor?\nExamples: A1, A1:B5", []string{"Cancel", TABS_STR})
		return
	}
	msg := "Send the number of the tab. Available tabs:\n"
	for i, name := range names {
		msg += "\n" + strconv.Itoa(i + 1) + ". " + name
	}
	messageChan <- makeMessage(uid, msg, []string{"Cancel", TABS_STR})
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
			return makeMessage(id, "Enter the cell URL. You can get it by right-clicking the cell and copying the link to it. " +
				"You may also just paste the table URL here and select the cell later.", []string{"Cancel"})
		case MENU_KB[1]:
			pairs := recordList(id)
			return makeMessageInline(id, formatRecordList(id, pairs), []string{"Edit", "Delete"})
		default:
			return makeMessage(id, "Wat?", MENU_KB)
		}
	case "add":
		message = strings.Trim(message, " ")
		parsed := parseURL(message)
		if len(parsed) != DATA_LENGTH {
			return makeMessage(id, "Invalid url, try again.", []string{"Cancel"})
		}
		data, err := json.Marshal(parsed)
		if err != nil {
			return makeMessage(id, "Something went wrong", []string{"Cancel"})
		}
		ustate["record"] = string(data)
		if parsed[2] == "" {
			ustate["name"] = "add-page"
			go sendPageList(id, parsed[0])
			return nil
		}
		ustate["name"] = "add-name"
		return makeMessage(id, "Enter the name for this cell", []string{"Cancel"})
	case "add-page":
		if message != TABS_STR {
			message = strings.Trim(message, " ")
			number, err := strconv.ParseInt(message, 10, 64)
			if err != nil || number < 0 {
				return makeMessage(id, "Bad number, try again", []string{"Cancel"})
			}
			data := parseList(ustate["record"])
			table := getTable(data[0])
			if table == nil {
				return makeMessage(id, "Could not fetch table, try again", []string{"Cancel"})
			}
			_, gids := getPageList(*table)
			if number > int64(len(gids)) {
				return makeMessage(id, "Bad number, try again", []string{"Cancel"})
			}
			data[1] = gids[number-1]
			cdata, _ := json.Marshal(data)
			ustate["record"] = string(cdata)
			ustate["name"] = "add-cell"
			return makeMessage(id, "What cell do you want to monitor?\nExamples: A1, A1:B5", []string{"Cancel", TABS_STR})
		}
		fallthrough
	case "add-cell":
		var parsed []string
		if message == TABS_STR {
			parsed = make([]string, 0)
			parsed = append(parsed, "", "tabs", "", "", "")
		} else {
			message = strings.ToUpper(strings.Trim(message, " "))
			parsed = CELL_RE.FindStringSubmatch(message)
			if len(parsed) != 5 {
				return makeMessage(id, "Invalid cell, try again.", []string{"Cancel"})
			}
		}
		data := parseList(ustate["record"])
		data[2] = parsed[1]
		data[3] = parsed[2]
		data[4] = parsed[3]
		data[5] = parsed[4]
		if data[4] == "" {
			data[4] = data[2]
		}
		if data[5] == "" {
			data[5] = data[3]
		}
		cdata, _ := json.Marshal(data)
		ustate["record"] = string(cdata)
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
		pairs := recordList(id)
		for _, val := range numbers {
			if val == "" {
				continue
			}
			res, err := strconv.ParseInt(val, 10, 64)
			if err != nil || res <= 0 || res > int64(len(pairs)) {
				return makeMessage(id, "Bad number, try again", []string{"Cancel"})
			}
			ints = append(ints, res)
		}
		for _, num := range ints {
			deleteRecord(id, pairs[num-1].Name)
			deleteCellVal(id, pairs[num-1].Name)
		}
		ustate["name"] = ""
		return makeMessage(id, "Deleted!", MENU_KB)
	case "edit":
		message = strings.Trim(message, " ")
		pairs := recordList(id)
		num, err := strconv.ParseInt(message, 10, 64)
		if err != nil || num <= 0 || num > int64(len(pairs)) {
			return makeMessage(id, "Bad number, try again", []string{"Cancel"})
		}
		ustate["record-name"] = pairs[num-1].Name
		ustate["record"] = pairs[num-1].Value
		ustate["name"] = "edit-cell"
		plist := parseList(ustate["record"])
		currentCell := plist[2] + plist[3]
		if plist[4] != plist[2] || plist[5] != plist[3] {
			currentCell += ":" + plist[4] + plist[5]
		}
		return makeMessage(id, "What cell do you want to monitor?\nCurrently: " + currentCell, []string{"Cancel", TABS_STR})
	case "edit-cell":
		var parsed []string
		if message == TABS_STR {
			parsed = make([]string, 0)
			parsed = append(parsed, "", "tabs", "", "", "")
		} else {
			message = strings.ToUpper(strings.Trim(message, " "))
			parsed = CELL_RE.FindStringSubmatch(message)
			if len(parsed) != 5 {
				return makeMessage(id, "Invalid cell, try again.", []string{"Cancel"})
			}
		}
		data := parseList(ustate["record"])
		data[2] = parsed[1]
		data[3] = parsed[2]
		data[4] = parsed[3]
		data[5] = parsed[4]
		if data[4] == "" {
			data[4] = data[2]
		}
		if data[5] == "" {
			data[5] = data[3]
		}
		cdata, _ := json.Marshal(data)
		deleteCellVal(id, ustate["record-name"])
		go sendInitialValue(id, data)
		deleteRecord(id, ustate["record-name"])
		addRecord(id, ustate["record-name"], string(cdata))
		ustate["name"] = ""
		return nil
	}
	return makeMessage(id, "Not implemented yet", MENU_KB)
}

func handleCallback(id int64, data string) *tgbotapi.MessageConfig {
	ustate, ok := state[id]
	if !ok {
		state[id] = make(map[string]string)
		ustate = state[id]
	}
	if data == "Delete" {
		pairs := recordList(id)
		if len(pairs) == 0 {
			ustate["name"] = ""
			return makeMessage(id, "You have no cells yet", MENU_KB)
		}
		ustate["name"] = "delete"
		return makeMessage(id, "Which cells do you want to delete?\n" +
			"Enter their numbers separated by commas or spaces", []string{"Cancel"})
	}
	if data == "Edit" {
		pairs := recordList(id)
		if len(pairs) == 0 {
			ustate["name"] = ""
			return makeMessage(id, "You have no cells yet", MENU_KB)
		}
		ustate["name"] = "edit"
		return makeMessage(id, "Which cell do you want to edit? Enter its number", []string{"Cancel"})
	}
	return makeMessage(id, "Not implemented yet", MENU_KB)
}