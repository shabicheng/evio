package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cihub/seelog"
	"github.com/shabicheng/evio/util"
)

// do not use this logger directly
var InnerLogtailLogger seelog.LoggerInterface
var DebugFlag bool

var defaultConfig = `
<seelog type="asynctimer" asyncinterval="500000" minlevel="info" >
 <outputs formatid="common">
	 <rollingfile type="size" filename="%slogtail_plugin.LOG" maxsize="2097152" maxrolls="10"/>
 </outputs>
 <formats>
	 <format id="common" format="%%Date %%Time [%%LEV] [%%File:%%Line] [%%FuncShort] %%Msg%%n" />
 </formats>
</seelog>
`

func generateLog(kvPairs ...interface{}) string {
	var logString = ""
	pairLen := len(kvPairs) / 2
	for i := 0; i < pairLen; i++ {
		logString += fmt.Sprintf("%v:%v\t", kvPairs[i<<1], kvPairs[i<<1+1])
	}
	if len(kvPairs)&0x01 != 0 {
		logString += fmt.Sprintf("%v:\t", kvPairs[len(kvPairs)-1])
	}
	return logString
}

func Debug(kvPairs ...interface{}) {
	if !DebugFlag {
		return
	}
	InnerLogtailLogger.Debug(generateLog(kvPairs...))
}

func Info(kvPairs ...interface{}) {
	InnerLogtailLogger.Info(generateLog(kvPairs...))
}

func Warning(alarmType string, kvPairs ...interface{}) {
	alarmMsg := generateLog(kvPairs...)
	InnerLogtailLogger.Warn("AlarmType:", alarmType, "\t", alarmMsg)
}

func Error(alarmType string, kvPairs ...interface{}) {
	alarmMsg := generateLog(kvPairs...)
	InnerLogtailLogger.Error("AlarmType:", alarmType, "\t", alarmMsg)
}

func setLogConf(logConfig string) {
	DebugFlag = false
	var err error
	InnerLogtailLogger = seelog.Disabled
	if _, err := os.Stat(logConfig); err != nil {
		logConfigContent := fmt.Sprintf(defaultConfig, util.GetCurrentBinaryPath())
		ioutil.WriteFile(logConfig, []byte(logConfigContent), os.ModePerm)
	}
	fmt.Printf("load log config %s \n", logConfig)
	InnerLogtailLogger, err = seelog.LoggerFromConfigAsFile(logConfig)
	if InnerLogtailLogger == nil {
		InnerLogtailLogger = seelog.Disabled
	} else {
		InnerLogtailLogger.SetAdditionalStackDepth(1)
	}
	if err != nil {
		fmt.Println("init logger error", err)
		return
	}
	dat, err := ioutil.ReadFile(logConfig)
	if err != nil {
		return
	}
	if strings.Contains(string(dat), "minlevel=\"debug\"") {
		DebugFlag = true
	}
}

func init() {
	setLogConf(util.GetCurrentBinaryPath() + "plugin_logger.xml")
}
