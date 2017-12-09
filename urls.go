package main

import (
	"regexp"
	"net/http"
	"log"
	"io/ioutil"
	"strings"
)

var URL_RE, _ = regexp.Compile(`https?://docs.google.com/spreadsheets/(.*)/(?:edit|htmlview|(pubhtml))(?:\?[^#]*)?#?(?:gid=(\d*)(?:&range=([A-Z]+)(\d+)(?:\:([A-Z]+)(\d+))?)?)?$`)
var CELL_RE, _ = regexp.Compile(`^([A-Z]+)(\d+)(?:\:([A-Z]+)(\d+))?$`)

const DATA_LENGTH = 6

func parseURL(url string) []string {
	res := URL_RE.FindStringSubmatch(url)
	if len(res) > 0 {
		res = res[1:]
		if res[1] == "pubhtml" {
			res[0] += "/pubhtml"
		}
		res = append(res[:1], res[2:]...)
		if res[4] == "" {
			res[4] = res[2]
		}
		if res[5] == "" {
			res[5] = res[3]
		}
		return res
	} else {
		return nil
	}
}

func buildEditURL(data []string) string {
	if len(data) != DATA_LENGTH {
		return ""
	}
	if strings.HasSuffix(data[0], "/pubhtml") {
		return "https://docs.google.com/spreadsheets/" + data[0] + " gid=" + data[1] + " range=" + data[2] + data[3] + ":" + data[4] + data[5]
	}
	if data[2] == "tabs" {
		return "https://docs.google.com/spreadsheets/" + data[0] + "/edit, tabs"
	}
	return "https://docs.google.com/spreadsheets/" + data[0] + "/edit#gid=" + data[1] + "&range=" + data[2] + data[3] + ":" + data[4] + data[5]
}

func fetchTable(name string) *string {
	var url string;
	if strings.HasSuffix(name, "/pubhtml") {
		url = "https://docs.google.com/spreadsheets/" + name
	} else {
		url = "https://docs.google.com/spreadsheets/" + name + "/htmlview"
	}
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		log.Println("Unable to fetch " + name)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	result := string(body)
	return &result
}
