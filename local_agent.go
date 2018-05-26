package main

func LocalAgentServer(loops, port int) {
	workerQueue := make(chan *AgentRequest, 100)
	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {
				req.conn.Send(AppendResp(nil, "200", "", "Hello world."))
				//log.Printf("get \n")
			}
		}()
	}

	ServeListenAgent(loops, port, workerQueue)
}
