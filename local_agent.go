package main

// provider 监听的本地Agent Server
// 接收到数据后，请求本地的dubbo agent

func LocalAgentServer(loops, port int) {
	workerQueue := make(chan *AgentRequest, 100)
	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {
				//req.conn.Send(AppendResp(nil, "200", "", "Hello world."))
				//logger.Info("get agent request id, detail %v", req.RequestID, *req)
				//SendAgentRequest(req.conn, 200, req.RequestID, req.Interf, req.Method, req.ParamType, []byte("Hello World.!~"))
				GlobalLocalDubboAgent.SendDubboRequest(req)
			}
		}()
	}

	ServeListenAgent(loops, port, workerQueue)
}
