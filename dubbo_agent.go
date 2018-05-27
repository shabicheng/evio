package main

import (
	"flag"
	"sync"
	"time"

	"github.com/shabicheng/evio/logger"

	dubbocodec "github.com/shabicheng/evio/codec"
	dubboagent "github.com/shabicheng/evio/codec/jsonrpc"
)

var dubboPort = flag.Int("dubbo-port", 20889, "")
var dubboHost = flag.String("dubbo-host", "127.0.0.1", "")
var dubboConnCount = flag.Int("dubbo-conn", 4, "")

var GlobalLocalDubboAgent LocalDubboAgent

type LocalDubboAgent struct {
	requestMap      sync.Map // key id, val agent request
	server          *server
	workerRespQueue chan *DubboReponse

	connManager ConnectionManager
}

type DubboReponse dubboagent.Response

type DubboContext struct {
	is InputStream
}

func CreateDubboEvent(loops int, workerQueue chan *DubboReponse) *Events {
	events := &Events{}
	events.NumLoops = loops
	events.Serving = func(srv Server) (action Action) {
		logger.Info("dubbo agent server started (loops: %d)", srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		lastCtx := c.Context()
		if lastCtx == nil {
			c.SetContext(&DubboContext{})
		}

		logger.Info("dubbo agent opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	}

	// producer 向dubbo 发起的链接，在close的时候需要销毁
	// 删除dubbo的连接
	events.Closed = func(c Conn, err error) (action Action) {
		logger.Info("dubbo agent closed: %s: %s", c.LocalAddr(), c.RemoteAddr())
		GlobalLocalDubboAgent.connManager.DeleteConnection(c)
		GlobalLocalDubboAgent.GetConnection()
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		//logger.Info("Data: laddr: %v: raddr: %v, data", c.LocalAddr(), c.RemoteAddr(), string(in))
		if in == nil {
			return
		}
		agentContext := c.Context().(*DubboContext)

		data := agentContext.is.Begin(in)
		// process the pipeline
		for {
			//logger.Info("data %v\n", data)
			leftover, resp, err := dubboagent.UnpackResponse(data)
			//logger.Info("result %v %v %v \n", leftover, err, ready)
			if err != nil {
				if err == dubbocodec.ErrHeaderNotEnough {
					// request not ready, yet
					data = leftover
					break
				} else {
					// bad thing happened
					action = Close
					break
				}
			}
			//fmt.Printf("insert %v \n", *resp)
			//AppendRequest(out, httpContext.req)
			workerQueue <- (*DubboReponse)(resp)
			// handle the request
			data = leftover
		}
		agentContext.is.End(data)
		return
	}
	return events
}

func (lda *LocalDubboAgent) ServeConnectDubbo(loops int) error {

	lda.workerRespQueue = make(chan *DubboReponse, 1000)

	for i := 0; i < 8; i++ {
		go func() {
			for resp := range lda.workerRespQueue {
				//fmt.Printf("insert %v \n", *resp)
				obj, ok := GlobalLocalDubboAgent.requestMap.Load(uint64(resp.ID))
				if !ok {
					logger.Info("receive dubbo client's response, but no this req id %d", int(resp.ID))
					continue
				}
				//logger.Info("receive dubbo client's response, ", int(resp.ID), string(resp.Data))
				agentReq := obj.(*AgentRequest)
				SendAgentRequest(agentReq.conn, 200, agentReq.RequestID, agentReq.Interf, agentReq.Method, agentReq.ParamType, resp.Data)
			}
		}()
	}

	events := CreateDubboEvent(4, lda.workerRespQueue)
	var err error
	lda.server, err = ConnServe(*events)
	return err
}

func (lda *LocalDubboAgent) GetConnection() Conn {
	resultConn, connCount := lda.connManager.GetConnection()
	createConnCount := *dubboConnCount - connCount

	makeConn := func() (Conn, error) {
		conn, err := outConnect(lda.server, *dubboHost, *dubboPort, &DubboContext{})
		if err != nil {
			logger.Warning("CONNECT_DUBBO_ERROR", err, *dubboPort, *dubboPort)
			return nil, err
		}

		lda.connManager.AddConnection(conn)
		return conn, nil
	}

	if connCount == 0 {
		for true {
			conn, err := makeConn()
			if err == nil {
				resultConn = conn
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		createConnCount--
	}

	for createConnCount > 0 {
		go makeConn()
		createConnCount--
	}

	return resultConn.(Conn)
}
