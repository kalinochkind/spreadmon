package main

import (
	"regexp"
	"net/http"
	"log"
	"io/ioutil"
)

var URL_RE, _ = regexp.Compile(`https?://docs.google.com/spreadsheets/(.*)/edit#gid=(\d*)&range=([A-Z]+)(\d+)`)

func parseURL(url string) []string {
	res := URL_RE.FindStringSubmatch(url)
	if len(res) > 0 {
		return res[1:]
	} else {
		return nil
	}
}

func buildEditURL(data []string) string {
	if len(data) != 4 {
		return ""
	}
	return "https://docs.google.com/spreadsheets/" + data[0] + "/edit#gid=" + data[1] + "&range=" + data[2] + data[3]
}

func fetchTable(name string) string {
	resp, err := http.Get("https://docs.google.com/spreadsheets/" + name + "/htmlview")
	if err != nil {
		log.Println("Unable to fetch " + name + ": " + err.Error())
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body)
}
