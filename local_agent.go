package main

import "github.com/shabicheng/evio/logger"

func LocalAgentServer(loops, port int) {
	workerQueue := make(chan *AgentRequest, 100)
	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {
				//req.conn.Send(AppendResp(nil, "200", "", "Hello world."))
				logger.Info("get agent request %v\n", *req)
				SendAgentRequest(req.conn, 200, req.RequestID, req.Interf, req.Method, req.ParamType, []byte("Hello World.!~"))
			}
		}()
	}

	ServeListenAgent(loops, port, workerQueue)
}
