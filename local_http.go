package main

func LocalHttpServer(loops, port int) {

	workerQueue := make(chan *HttpRequest, 100)

	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {

				agentReq := &AgentRequest{
					Interf:    GlobalInterface,
					Method:    req.callMethod,
					ParamType: ParamType_String,
					Param:     []byte(req.parameter),
				}
				GlobalRemoteAgentManager.ForwardRequest(agentReq, req)
				//req.conn.Send(AppendResp(nil, "200", "", "Hello world."))
				//log.Printf("get \n")
			}
		}()
	}

	ServeListenHttp(loops, port, workerQueue)
}
