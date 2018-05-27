package main

import "github.com/shabicheng/evio/logger"

func LocalHttpServer(loops, port int) {

	workerQueue := make(chan *HttpRequest, 100)

	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {

				agentReq := &AgentRequest{
					Interf:    req.interf,
					Method:    req.callMethod,
					ParamType: ParamType_String,
					Param:     []byte(req.parameter),
				}
				logger.Info("req.interf", req.interf, req.callMethod)
				GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
				//req.conn.Send(AppendResp(nil, "200", "", "Hello world."))
				//logger.Info("get \n")
			}
		}()
	}

	ServeListenHttp(loops, port, workerQueue)
}
