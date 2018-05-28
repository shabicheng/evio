package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	//_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"time"
)

var localLoops = flag.Int("local-loop", 1, "local loop count")
var localPort = flag.Int("local-port", 20000, "local loop count")
var mode = flag.String("mode", "consumer", "mode")
var providerPort = flag.Int("provider-port", 30000, "provide agent listen port")
var providerLoops = flag.Int("provider-loop", 1, "provide loop count")
var defaultAgentCount = flag.Int("agent-count", 4, "default agent connection count")

func DumpCpuInfo(seconds int) {
	f, err := os.Create(*mode + "_cpuprof.prof")
	if err != nil {
		return
	}
	pprof.StartCPUProfile(f)
	time.Sleep((time.Duration)(seconds) * time.Second)
	pprof.StopCPUProfile()
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	flag.Parse()

	go DumpCpuInfo(60)
	go func() {
		if *mode == "consumer" {
			log.Println(http.ListenAndServe("localhost:8088", nil))
		} else {
			log.Println(http.ListenAndServe("localhost:8089", nil))
		}
	}()

	if *mode == "consumer" {
		GlobalRemoteAgentManager.ServeConnectAgent()
		go GlobalRemoteAgentManager.ListenInterface(GlobalInterface)

		// time.Sleep(time.Second)
		// req := &HttpRequest{
		// 	callMethod: "hash",
		// 	parameter:  "xxxxxxxxxxx",
		// }

		// agentReq := &AgentRequest{
		// 	Interf:    "com.alibaba.dubbo.performance.demo.provider.IHelloService",
		//              com.alibaba.dubbo.performance.demo.provider.IHelloService
		// 	Method:    req.callMethod,
		// 	ParamType: ParamType_String,
		// 	Param:     []byte(req.parameter),
		// }

		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		// time.Sleep(time.Second)

		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
		// GlobalRemoteAgentManager.ForwardRequest(agentReq, req)

		time.Sleep(time.Second)

		LocalHttpServer(*localLoops, *localPort)
	} else {

		GlobalLocalDubboAgent.ServeConnectDubbo(4)
		time.Sleep(time.Second)
		go GlobalRemoteAgentManager.RegisterInterface(GlobalInterface, *providerPort)

		LocalAgentServer(*providerLoops, *providerPort)
	}
}
