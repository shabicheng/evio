package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"time"

	"github.com/cihub/seelog"
	"github.com/shabicheng/evio/logger"
)

var localPort = flag.Int("local-port", 20000, "local loop count")
var mode = flag.String("mode", "consumer", "mode")
var providerPort = flag.Int("provider-port", 30000, "provide agent listen port")
var defaultAgentCount = flag.Int("agent-count", 4, "default agent connection count")
var profileDir = flag.String("profile-dir", "./", "profile dir, set to /root/logs/")

// loop 设置
var consumerHttpLoops = flag.Int("consumer-http-loop", 2, "")
var consumerAgentLoops = flag.Int("consumer-agent-loop", 1, "")
var providerAgentLoops = flag.Int("provider-agent-loop", 1, "")
var providerDubboLoops = flag.Int("provider-dubbo-loop", 2, "")

// processor 设置
var consumerHttpProcessors = flag.Int("consumer-http-processor", 4, "")
var consumerAgentProcessors = flag.Int("consumer-agent-processor", 4, "")
var providerAgentProcessors = flag.Int("provider-agent-processor", 4, "")
var providerDubboProcessors = flag.Int("provider-dubbo-processor", 4, "")

var ProfileLogger seelog.LoggerInterface

func DumpCpuInfo(seconds int) {
	f, err := os.Create(*profileDir + *mode + "_cpuprof.prof")
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

	ProfileLogger = logger.GetProfileLogger(*mode, *profileDir)

	go DumpCpuInfo(60)
	go func() {
		if *mode == "consumer" {
			log.Println(http.ListenAndServe("localhost:8088", nil))
		} else {
			log.Println(http.ListenAndServe("localhost:8089", nil))
		}
	}()

	if *mode == "consumer" {
		GlobalRemoteAgentManager.ServeConnectAgent(*consumerAgentLoops)
		go GlobalRemoteAgentManager.ListenInterface(GlobalInterface)

		time.Sleep(time.Second)

		LocalHttpServer(*consumerHttpLoops, *localPort)
	} else {

		GlobalLocalDubboAgent.ServeConnectDubbo(*providerDubboLoops)
		time.Sleep(time.Second)
		go GlobalRemoteAgentManager.RegisterInterface(GlobalInterface, *providerPort)

		LocalAgentServer(*providerAgentLoops, *providerPort)
	}
}
