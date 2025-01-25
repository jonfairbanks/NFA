package main

import (
	"log"

	"github.com/MORpheusSoftware/NFA/BaseImage/proxy"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	proxy.StartProxyServer()
}