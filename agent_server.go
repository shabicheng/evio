package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
)

const (
	agentPhase_Header    = 0
	agentPhase_Interface = 1
	agentPhase_Method    = 2
	agentPhase_ParamType = 3
	agentPhase_Param     = 4

	agentPacketMagic = 0x1F254798
)

var agentPacketParseError = errors.New("agentPacketParseError")

type ParamType int

const (
	ParamType_None = iota
	ParamType_Int8
	ParamType_Int16
	ParamType_Int32
	ParamType_Uint8
	ParamType_Uint16
	ParamType_Uint32
	ParamType_String
	ParamType_Obj
)

type AgentContext struct {
	ra  *RemoteAgent
	is  InputStream
	req *AgentRequest
}

type AgentRequest struct {
	conn     Conn
	paramLen int
	phase    int

	RequestID  uint64
	Result     uint32
	RemoteAddr string
	Interf     string
	Method     string
	ParamType  ParamType
	Param      []byte
}

func CreateAgentEvent(loops int, workerQueue chan *AgentRequest) *Events {
	events := &Events{}
	events.NumLoops = loops
	events.Serving = func(srv Server) (action Action) {
		log.Printf("agent server started (loops: %d)", srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		lastCtx := c.Context()
		if lastCtx == nil {
			c.SetContext(&AgentContext{})
		}

		log.Printf("agent opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	}

	events.Closed = func(c Conn, err error) (action Action) {
		log.Printf("agent closed: %s: %s", c.LocalAddr().String(), c.RemoteAddr().String())
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		//log.Printf("Data: laddr: %v: raddr: %v, data", c.LocalAddr(), c.RemoteAddr(), string(in))
		if in == nil {
			return
		}
		agentContext := c.Context().(*AgentContext)

		if agentContext.req == nil {
			agentContext.req = &AgentRequest{}
			agentContext.req.conn = c
			agentContext.is.b = nil
			// handle the request
			agentContext.req.RemoteAddr = c.RemoteAddr().String()
		}
		data := agentContext.is.Begin(in)
		// process the pipeline
		for {
			leftover, err, ready := parseAgentReq(data, agentContext.req)
			//log.Printf("result %v %v %v \n", leftover, err, ready)
			if err != nil {
				// bad thing happened
				action = Close
				break
			} else if !ready {
				// request not ready, yet
				data = leftover
				break
			}
			//log.Printf("insert \n")
			//AppendRequest(out, httpContext.req)
			workerQueue <- agentContext.req
			agentContext.req = nil
			agentContext.req = &AgentRequest{}
			agentContext.req.conn = c
			agentContext.is.b = nil
			// handle the request
			agentContext.req.RemoteAddr = c.RemoteAddr().String()
			data = leftover
		}
		agentContext.is.End(data)
		return
	}
	return events
}

func ServeListenAgent(loops int, port int, workerQueue chan *AgentRequest) error {
	events := CreateAgentEvent(loops, workerQueue)
	// We at least want the single http address.
	addrs := []string{fmt.Sprintf("tcp://:%d?reuseport=true", port)}
	// Start serving!
	return Serve(*events, addrs...)
}

func SendAgentRequest(conn Conn,
	result uint32,
	reqID uint64,
	interf string,
	method string,
	paramType ParamType,
	param []byte) (err error) {
	out := make([]byte, 0, 26+len(interf)+len(method)+len(param))

	binary.LittleEndian.PutUint32(out, uint32(agentPacketMagic))
	buf := out[4:]
	binary.LittleEndian.PutUint32(buf, result)
	buf = out[8:]
	binary.LittleEndian.PutUint64(buf, reqID)
	buf = out[4:]
	binary.LittleEndian.PutUint16(buf, uint16(len(interf)))
	buf = out[2:]
	buf = append(buf, interf...)
	buf = out[len(interf):]
	binary.LittleEndian.PutUint16(buf, uint16(len(method)))
	buf = out[2:]
	buf = append(buf, method...)
	buf = out[len(method):]
	binary.LittleEndian.PutUint16(buf, uint16(paramType))
	buf = out[2:]

	binary.LittleEndian.PutUint32(buf, uint32(len(param)))
	buf = out[4:]
	buf = append(buf, param...)
	return conn.Send(buf)

}

func parseAgentReq(data []byte, req *AgentRequest) (body []byte, err error, ready bool) {
	body = data
	for {
		switch req.phase {
		case agentPhase_Header:
			if len(body) < 16 {
				return body, err, false
			}
			// @todo parse header
			magic := binary.LittleEndian.Uint32(body)
			if magic != agentPacketMagic {
				return body, agentPacketParseError, false
			}
			req.Result = binary.LittleEndian.Uint32(body[4:])
			req.RequestID = binary.LittleEndian.Uint64(body[8:])
			req.phase = agentPhase_Interface
			body = body[8:]
		case agentPhase_Interface:
			if len(body) < 2 {
				return body, err, false
			}
			interfaceLen := binary.LittleEndian.Uint16(body)
			if len(body)-2 < int(interfaceLen) {
				return body, err, false
			}
			req.Interf = string(body[2:(2 + interfaceLen)])
			body = body[2+interfaceLen:]
			req.phase = agentPhase_Method
		case agentPhase_Method:
			if len(body) < 2 {
				return body, err, false
			}
			methodLen := binary.LittleEndian.Uint16(body)
			if len(body)-2 < int(methodLen) {
				return body, err, false
			}
			req.Method = string(body[2:(2 + methodLen)])
			body = body[2+methodLen:]
			req.phase = agentPhase_Param
		case agentPhase_Param:
			if len(body) < 6 {
				return body, err, false
			}
			req.ParamType = ParamType(binary.LittleEndian.Uint16(body))
			paramLen := binary.LittleEndian.Uint16(body[2:])
			if len(body)-6 < int(paramLen) {
				return body, err, false
			}
			req.Param = body[6:(6 + paramLen)]
			body = body[6+paramLen:]
			return body, nil, true
		}
	}
}
