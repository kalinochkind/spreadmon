package main

import "flag"

var configMap = make(map[string]string)

func init() {
	token := flag.String("token", "", "token for telegram")
	addr := flag.String("addr", "localhost:6379", "redis address")
	passwd := flag.String("passwd", "", "redis password")
	flag.Parse()
	configMap["token"] = *token
	configMap["addr"] = *addr
	configMap["passwd"] = *passwd
}
