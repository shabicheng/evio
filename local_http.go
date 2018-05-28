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
				//logger.Info("req.interf", req.interf, req.callMethod)
				if err := GlobalRemoteAgentManager.ForwardRequest(agentReq, req); err != nil {
					logger.Warning("forward request error", err)
					req.conn.Send(AppendResp(nil, "500", err.Error(), "forward request error"))
				}
				//req.conn.Send(AppendResp(nil, "200", "", "Hello world."))

			}
		}()
	}

	ServeListenHttp(loops, port, workerQueue)
}
