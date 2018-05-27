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
		go GlobalRemoteAgentManager.ListenInterface(GlobalInterface)
		time.Sleep(time.Second)
		req := &HttpRequest{
			callMethod: "12345",
			parameter:  "xxxx",
		}

		agentReq := &AgentRequest{
			Interf:    GlobalInterface,
			Method:    req.callMethod,
			ParamType: ParamType_String,
			Param:     []byte(req.parameter),
		}

		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)

		time.Sleep(time.Second)

		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)

		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		GlobalRemoteAgentManager.ForwardRequest(agentReq, req)

		LocalHttpServer(*localLoops, *localPort)
	} else {
		go GlobalRemoteAgentManager.RegisterInterface(GlobalInterface, *providerPort)
		LocalAgentServer(*providerLoops, *providerPort)
	}
}
