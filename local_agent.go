package main

import (
	dubbocodec "github.com/shabicheng/evio/codec"
	dubbojson "github.com/shabicheng/evio/codec/jsonrpc"
	"github.com/shabicheng/evio/logger"
)

func LocalAgentServer(loops, port int) {
	workerQueue := make(chan *AgentRequest, 100)
	for i := 0; i < loops; i++ {
		go func() {
			for req := range workerQueue {
				//req.conn.Send(AppendResp(nil, "200", "", "Hello world."))
				logger.Info("get agent request id, detail %v\n", req.RequestID, *req)
				//SendAgentRequest(req.conn, 200, req.RequestID, req.Interf, req.Method, req.ParamType, []byte("Hello World.!~"))

				attachments := make(map[string]string)
				attachments["path"] = req.Interf
				// send to dubbo agent
				message := &dubbocodec.Message{
					ID:        int64(req.RequestID),
					Version:   "2.6.0",
					Type:      dubbocodec.Request,
					Interface: req.Interf, // Service
					Method:    req.Method,
					Data: &dubbocodec.RpcInvocation{
						Method:         req.Method,
						ParameterTypes: req.ParamType.String(),
						Args:           req.Param,
						Attachments:    attachments,
					},
				}

				data, e := dubbojson.PackRequest(message)
				if e != nil {
					// do something
				}
				GlobalLocalDubboAgent.GetConnection().Send(data)
			}
		}()
	}

	ServeListenAgent(loops, port, workerQueue)
}
