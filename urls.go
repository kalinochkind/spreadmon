package main

import (
	"regexp"
	"net/http"
	"log"
	"io/ioutil"
	"strings"
)

var URL_RE, _ = regexp.Compile(`https?://docs.google.com/spreadsheets/(.*)/(?:edit|htmlview|(pubhtml))(?:\?[^#]*)?#?(?:gid=(\d*)(?:&range=([A-Z]+)(\d+))?)?$`)
var CELL_RE, _ = regexp.Compile(`^([A-Z]+)(\d+)$`)

func parseURL(url string) []string {
	res := URL_RE.FindStringSubmatch(url)
	if len(res) > 0 {
		res = res[1:]
		if res[1] == "pubhtml" {
			res[0] += "/pubhtml"
		}
		res = append(res[:1], res[2:]...)
		return res
	} else {
		return nil
	}
}

func buildEditURL(data []string) string {
	if len(data) != 4 {
		return ""
	}
	if strings.HasSuffix(data[0], "/pubhtml") {
		return "https://docs.google.com/spreadsheets/" + data[0]
	}
	return "https://docs.google.com/spreadsheets/" + data[0] + "/edit#gid=" + data[1] + "&range=" + data[2] + data[3]
}

func fetchTable(name string) string {
	var url string;
	if strings.HasSuffix(name, "/pubhtml") {
		url = "https://docs.google.com/spreadsheets/" + name
	} else {
		url = "https://docs.google.com/spreadsheets/" + name + "/htmlview"
	}
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		log.Println("Unable to fetch " + name)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body)
}
