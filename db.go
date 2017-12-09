package main

import (
	"github.com/go-redis/redis"
	"strconv"
	"encoding/json"
	"sync"
	"log"
	"sort"
)

var database *redis.Client

type StringPair struct {
	Name string
	Value string
}

type StringPairs []StringPair

func (s StringPairs) Len() int {
	return len(s)
}

func (s StringPairs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s StringPairs) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func connect() {
	database = redis.NewClient(&redis.Options{Addr: configMap["addr"], Password: configMap["passwd"], DB: 0})
	_, err := database.Ping().Result()
	if err != nil {
		log.Panic(err.Error())
	}
}

func recordExists(uid int64, name string) bool {
	_, err := database.HGet("records/"+strconv.FormatInt(uid, 10), name).Result()
	return err == nil
}

func addRecord(uid int64, name string, record string) {
	database.HSet("records/"+strconv.FormatInt(uid, 10), name, record)
}

func parseList(l string) []string {
	var s []string
	json.Unmarshal([]byte(l), &s)
	return s
}

func recordList(uid int64) StringPairs {
	res := make(StringPairs, 0)
	for name, value := range database.HGetAll("records/" + strconv.FormatInt(uid, 10)).Val() {
		res = append(res, StringPair{name, value})
	}
	sort.Sort(res)
	return res
}

func deleteRecord(uid int64, name string) {
	database.HDel("records/"+strconv.FormatInt(uid, 10), name)
}

var tableCache = make(map[string]string)

var tableLock = sync.Mutex{}

func getTable(name string) *string {
	tableLock.Lock()
	defer tableLock.Unlock()
	data, ok := tableCache[name]
	if ok {
		return &data
	}
	pdata := fetchTable(name)
	if pdata != nil {
		tableCache[name] = *pdata
	}
	return pdata
}

func clearTableCache() {
	tableLock.Lock()
	for i := range tableCache {
		delete(tableCache, i)
	}
	tableLock.Unlock()
}

func userList() []int64 {
	l := database.Keys("records/*").Val()
	res := make([]int64, len(l))
	for i, v := range l {
		res[i], _ = strconv.ParseInt(v[8:], 10, 64)
	}
	return res
}

var cellLock = sync.Mutex{}

func getCellVal(uid int64, name string) (string, bool) {
	val, err := database.HGet("cells/"+strconv.FormatInt(uid, 10), name).Result()
	if err == nil {
		return val, true
	} else {
		return "", false
	}
}

func updateCellVal(uid int64, name string, value string) string {
	cellLock.Lock()
	defer cellLock.Unlock()
	old, err := database.HGet("cells/"+strconv.FormatInt(uid, 10), name).Result()
	if err != nil || old != value {
		database.HSet("cells/"+strconv.FormatInt(uid, 10), name, value)
		if err != nil {
			return value
		}
	}
	return old
}

func deleteCellVal(uid int64, name string) {
	cellLock.Lock()
	database.HDel("cells/"+strconv.FormatInt(uid, 10), name)
	cellLock.Unlock()
}
