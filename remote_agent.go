package main

import (
	"context"
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
)

var etcdHost = flag.String("etcd-host", "etcd", "")
var etcdPort = flag.Int("etcd-port", 2379, "")

// 服务注册的key为:    /dubbomesh/com.some.package.IHelloService/192.168.100.100:2000
var GlobalInterface = "/dubbomesh/com.alibaba.dubbo.performance.demo.provider.IHelloService/"

var GlobalRemoteAgentManager RemoteAgentManager

type RemoteAgent struct {
	connList      []Conn
	lastSendIndex int
	sendCount     int
	addr          string
	port          int
	allInterface  sync.Map
	requestMap    sync.Map // key requestID, val Request
	reqID         uint64
}

func (ra *RemoteAgent) processResponse(respQueue chan *AgentRequest) error {
	for resp := range respQueue {
		if val, ok := ra.requestMap.Load(resp.RequestID); ok {
			req := val.(Request)
			req.Response(resp)
		} else {
			// error
		}

	}
	return nil
}

func (ra *RemoteAgent) GetConnection() Conn {
	return nil
}

func (ra *RemoteAgent) AddInterface(interf string) {
	ra.allInterface.Store(interf, nil)
}

func (ra *RemoteAgent) CreateConnection(wg *sync.WaitGroup) Conn {
	return nil
}

func (ra *RemoteAgent) SendRequest(req *AgentRequest, httpReq *HttpRequest) error {
	conn := ra.GetConnection()
	req.RequestID = atomic.AddUint64(&ra.reqID, 1)
	ra.requestMap.Store(req.RequestID, httpReq)
	return SendAgentRequest(conn, req.Result, req.RequestID, req.Interf, req.Method, req.ParamType, req.Param)
}

type RemoteAgentManager struct {
	allAgents         sync.Map // key addr, value *RemoteAgent
	cacheInterfaceMap sync.Map // key interface, val []RemoteAgent
	hackAgents        []RemoteAgent
	hackLB            LoadBalancer
	workerRespQueue   chan *AgentRequest
	server            *server
}

// 简单负载均衡
type LoadBalancer struct {
	weight    []uint32
	minWeight []uint32
	nowBase   uint32

	lastCount     uint32
	lastIndex     uint32
	lastBaseCount uint32
}

func (lb *LoadBalancer) Get() int {
	newCount := atomic.AddUint32(&lb.lastCount, 1)
	if newCount-atomic.LoadUint32(&lb.lastBaseCount) >
		lb.minWeight[atomic.LoadUint32(&lb.lastIndex)%uint32(len(lb.weight))] {
		newIndex := atomic.AddUint32(&lb.lastIndex, 1)
		atomic.StoreUint32(&lb.lastBaseCount, newCount)
		return int(newIndex)
	}
	return int(atomic.LoadUint32(&lb.lastIndex))
}

func (lb *LoadBalancer) Update(index, weight uint32) {
	if len(lb.weight) <= int(index) {
		lb.weight = append(lb.weight, weight)
		if index == 0 {
			lb.minWeight = append(lb.minWeight, weight/100)
			lb.nowBase = 100
		} else {
			lb.minWeight = append(lb.minWeight, weight/lb.nowBase)
		}
	} else {
		atomic.StoreUint32(&lb.weight[index], weight)
		atomic.StoreUint32(&lb.minWeight[index], weight/lb.nowBase)
	}
}

func (ram *RemoteAgentManager) AddAgent(addr string, port int, interf string, defaultConn int, weight int) error {
	agent, ok := ram.allAgents.Load(addr)
	if ok {
		ra := agent.(*RemoteAgent)
		ra.AddInterface(interf)
		return nil
	}
	ra := RemoteAgent{
		addr: addr,
		port: port,
	}
	ra.AddInterface(interf)
	for i := 0; i < defaultConn; i++ {
		go ra.CreateConnection(nil)
	}
	return nil
}

func (ram *RemoteAgentManager) getInterfaceKey(interf string, port int) string {
	return interf + util.GetIpAddress() + ":" + strconv.Itoa(port)
}

func (ram *RemoteAgentManager) ForwardRequest(agentReq *AgentRequest, httpReq *HttpRequest) error {
	// 老夫先hack一把
	if len(ram.hackAgents) == 0 {
		return nil
	}
	ram.hackAgents[ram.hackLB.Get()].SendRequest(agentReq, httpReq)
	return nil
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

	_, err = cli.Put(context.TODO(), ram.getInterfaceKey(interf, port), "", etcd.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}

	// the key will be kept forever
	ch, kaerr := cli.KeepAlive(context.TODO(), resp.ID)
	if kaerr != nil {
		log.Fatal(kaerr)
	}

	ka := <-ch
	fmt.Println("ttl:", ka.TTL)
}

func (ram *RemoteAgentManager) ServeConnectAgent() error {
	ram.workerRespQueue = make(chan *AgentRequest, 1000)

	for i := 0; i < 8; i++ {
		go func() {
			for resp := range ram.workerRespQueue {
				ctx := resp.conn.Context().(*AgentContext)
				obj, ok := ctx.ra.requestMap.Load(resp.RequestID)
				if !ok {
					log.Printf("receive remote agent's response, but no this req id %d", int(resp.RequestID))
				}
				httpReq := obj.(*HttpRequest)
				httpReq.Response(resp)
			}
		}()
	}

	events := CreateAgentEvent(4, ram.workerRespQueue)
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
			keyTotal := string(ev.Key)
			tail := keyTotal[strings.LastIndexByte(keyTotal, '/'):]
			keyValuePairs := strings.Split(tail, ":")
			addr := keyValuePairs[0]
			port, _ := strconv.Atoi(keyValuePairs[1])
			ram.AddAgent(addr, port, interf, 4, 1000)
		}
	}

	rch := c.Watch(context.Background(), "foo", etcd.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			fmt.Printf("etcd key events %s %q : %q\n", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value))

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
