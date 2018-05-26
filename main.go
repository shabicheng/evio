package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"time"
)

var localLoops = flag.Int("local-loop", 8, "local loop count")
var localPort = flag.Int("local-port", 20000, "local loop count")
var mode = flag.String("mode", "consumer", "mode")
var providerPort = flag.Int("provider-port", 30000, "provide agent listen port")
var providerLoops = flag.Int("provider-loop", 4, "provide loop count")

func main() {
	rand.Seed(int64(time.Now().UnixNano()))
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	flag.Parse()
	if *mode == "consumer" {
		GlobalRemoteAgentManager.ServeConnectAgent()
		time.Sleep(time.Second)
		outConnect(GlobalRemoteAgentManager.server, "127.0.0.1", *providerPort, nil)
		outConnect(GlobalRemoteAgentManager.server, "127.0.0.1", *providerPort, nil)
		outConnect(GlobalRemoteAgentManager.server, "127.0.0.1", *providerPort, nil)
		outConnect(GlobalRemoteAgentManager.server, "127.0.0.1", *providerPort, nil)
		outConnect(GlobalRemoteAgentManager.server, "127.0.0.1", *providerPort, nil)
		LocalHttpServer(*localLoops, *localPort)
	} else {
		LocalAgentServer(*providerLoops, *providerPort)
	}
}
