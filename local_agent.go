package main

// provider 监听的本地Agent Server
// 接收到数据后，请求本地的dubbo agent

func LocalAgentServer(loops, port int) {
	workerQueue := make(chan *AgentRequest, 100)
	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {
				//logger.Info("get agent request id, detail %v", req.RequestID, *req)
				//time.Sleep(time.Millisecond * 50)
				//SendAgentRequest(req.conn, 200, req.RequestID, req.Interf, req.Method, req.ParamType, []byte("Hello World.!~"))
				GlobalLocalDubboAgent.SendDubboRequest(req)
			}
		}()
	}

	ServeListenAgent(loops, port, workerQueue)
}
