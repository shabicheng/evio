package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shabicheng/evio/util"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/shabicheng/evio/logger"
)

var etcdHost = flag.String("etcd-host", "localhost", "")
var etcdPort = flag.Int("etcd-port", 2379, "")

var noAgentConnectionError = errors.New("noAgentConnectionError")
var noAgentError = errors.New("noAgentError")

// 服务注册的key为:    /dubbomesh/com.some.package.IHelloService/192.168.100.100:2000
var GlobalInterface = "/dubbomesh/com.alibaba.dubbo.performance.demo.provider.IHelloService/"

var GlobalRemoteAgentManager RemoteAgentManager

type RemoteAgent struct {
	connManager   ConnectionManager
	lastSendIndex int
	sendCount     uint64
	addr          string
	port          int
	allInterface  sync.Map
	requestMap    sync.Map // key requestID, val Request
	reqID         uint64
	defaultConn   int
}

func (ra *RemoteAgent) GetConnection() Conn {
	connInterf, connCount := ra.connManager.GetConnection()
	createConnCount := ra.defaultConn - connCount
	for createConnCount > 0 {
		go ra.CreateConnection(nil)
		createConnCount--
	}
	if connInterf != nil {
		return connInterf.(Conn)
	}
	conn, _ := ra.CreateConnection(nil)
	return conn
}

func (ra *RemoteAgent) AddInterface(interf string) {
	ra.allInterface.Store(interf, nil)
}

func (ra *RemoteAgent) CreateConnection(wg *sync.WaitGroup) (Conn, error) {
	conn, err := outConnect(GlobalRemoteAgentManager.server, ra.addr, ra.port, &AgentContext{ra: ra})
	if err == nil {
		ra.connManager.AddConnection(conn)
	}
	return conn, err
}

func (ra *RemoteAgent) SendRequest(req *AgentRequest, httpReq *HttpRequest) error {
	conn := ra.GetConnection()
	if conn == nil {
		return noAgentConnectionError
	}
	req.RequestID = atomic.AddUint64(&ra.reqID, 1)
	ra.requestMap.Store(req.RequestID, httpReq)
	return SendAgentRequest(conn, req.Result, req.RequestID, req.Interf, req.Method, req.ParamType, req.Param)
}

type RemoteAgentManager struct {
	allAgents         sync.Map // key addr, value *RemoteAgent
	cacheInterfaceMap sync.Map // key interface, val []RemoteAgent

	//hackAgents      []*RemoteAgent
	//hackLB          LoadBalancer
	hackManager     ConnectionManager
	workerRespQueue chan *AgentRequest
	server          *server
}

func (ram *RemoteAgentManager) AddAgent(addr string, port int, interf string, defaultConn int, weight int) error {
	existIdx := -1
	allConns := ram.hackManager.GetAllConnections()
	for idx, conn := range allConns {
		//fmt.Print(idx, " ", conn, "\n")
		agent := conn.(*RemoteAgent)
		if agent.addr == addr && agent.port == port {
			existIdx = idx
			break
		}
	}
	if existIdx >= 0 {
		logger.Info("update agent weight %s %s %d", addr, port, weight)
		ram.hackManager.UpdateLB(uint32(existIdx), uint32(weight))
		return nil
	}
	existIdx = len(allConns)
	logger.Info("create agent weight %s %s %d", addr, port, weight)
	ra := &RemoteAgent{
		addr:        addr,
		port:        port,
		defaultConn: defaultConn,
	}
	ra.AddInterface(interf)
	successCount := 0
	for i := 0; i < defaultConn; i++ {
		_, err := ra.CreateConnection(nil)
		if err == nil {
			successCount++
		} else {
			logger.Info("create agent weight %s %s %d", addr, port, weight)
		}
	}
	ram.hackManager.AddConnection(ra)

	ram.hackManager.UpdateLB(uint32(existIdx), uint32(weight))
	return nil

	// TODO 后面的功能还没写完，再议

	return nil
}

func (ram *RemoteAgentManager) getInterfaceKey(interf string, port int) string {
	return interf + util.GetHostName() + ":" + strconv.Itoa(port)
}

func (ram *RemoteAgentManager) ForwardRequest(agentReq *AgentRequest, httpReq *HttpRequest) error {
	// 老夫先hack一把
	conn, connCount := ram.hackManager.GetConnection()
	if connCount == 0 {
		logger.Info("empty agents \n")
		return noAgentError
	}
	return conn.(*RemoteAgent).SendRequest(agentReq, httpReq)
}

func (ram *RemoteAgentManager) RegisterInterface(interf string, port int) {
	cli, err := etcd.New(etcd.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", *etcdHost, *etcdPort)},
		DialTimeout: time.Second * 3,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	resp, err := cli.Grant(context.TODO(), 5)
	if err != nil {
		log.Fatal(err)
	}

	interfaceKey := ram.getInterfaceKey(interf, port)
	logger.Info("register interface to etcd", interfaceKey)
	_, err = cli.Put(context.TODO(), interfaceKey, "", etcd.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}

	// the key will be kept forever
	ch, kaerr := cli.KeepAlive(context.TODO(), resp.ID)
	if kaerr != nil {
		log.Fatal(kaerr)
	}
	select {}
	ka := <-ch
	logger.Info("ttl:", ka.TTL)
}

func (ram *RemoteAgentManager) DeleteRemoteAgentConnection(conn Conn) {
	agentCtx := conn.Context().(*AgentContext)
	deletedCount := agentCtx.ra.connManager.DeleteConnection(conn)

	agentCtx.ra.CreateConnection(nil)
	logger.Info("ServeConnectAgent agent closed", conn.LocalAddr(), conn.RemoteAddr(), "deleted count", deletedCount, "now count", agentCtx.ra.connManager.GetConnectionCount())
}

func (ram *RemoteAgentManager) ServeConnectAgent() error {
	ram.workerRespQueue = make(chan *AgentRequest, 1000)

	for i := 0; i < 8; i++ {
		go func() {
			for resp := range ram.workerRespQueue {
				ctx := resp.conn.Context().(*AgentContext)
				obj, ok := ctx.ra.requestMap.Load(resp.RequestID)
				if !ok {
					logger.Info("receive remote agent's response, but no this req id %d", int(resp.RequestID))
					continue
				}
				//logger.Info("receive remote agent's response, ", string(resp.Param))
				httpReq := obj.(*HttpRequest)
				httpReq.Response(resp)
				ctx.ra.requestMap.Delete(resp.RequestID)
			}
		}()
	}

	events := CreateAgentEvent(4, ram.workerRespQueue)
	events.Closed = func(c Conn, err error) (action Action) {
		ram.DeleteRemoteAgentConnection(c)

		return
	}
	var err error
	ram.server, err = ConnServe(*events)
	return err
}

func (ram *RemoteAgentManager) ListenInterface(interf string) {
	cfg := etcd.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", *etcdHost, *etcdPort)},
		DialTimeout: time.Second * 3,
	}
	c, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()
	if resp, err := c.Get(context.Background(), interf, etcd.WithPrefix()); err != nil {
		log.Fatal(err)
	} else {
		for _, ev := range resp.Kvs {
			logger.Info(fmt.Sprintf("etcd key events : ", ev.Key, string(ev.Key), string(ev.Value)))
			keyTotal := string(ev.Key)
			tail := keyTotal[strings.LastIndexByte(keyTotal, '/')+1:]
			keyValuePairs := strings.Split(tail, ":")
			addr := keyValuePairs[0]
			port, _ := strconv.Atoi(keyValuePairs[1])
			ram.AddAgent(addr, port, interf, *defaultAgentCount, 1000)
		}
	}

	rch := c.Watch(context.Background(), interf, etcd.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			logger.Info(fmt.Sprintf("etcd key events %s %q : %q", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value)))

			switch ev.Type {
			case mvccpb.PUT:
				keyTotal := string(ev.Kv.Key)
				tail := keyTotal[strings.LastIndexByte(keyTotal, '/'):]
				keyValuePairs := strings.Split(tail, ":")
				addr := keyValuePairs[0]
				port, _ := strconv.Atoi(keyValuePairs[1])
				weight, _ := strconv.Atoi(string(ev.Kv.Value))
				if weight > 0 {
					// @todo 更新Agent 能力
					ram.AddAgent(addr, port, interf, 4, weight)
				} else {
					ram.AddAgent(addr, port, interf, 4, 1000)
				}
			case mvccpb.DELETE:
				// @todo 删除Agent
			}
		}
	}
}
