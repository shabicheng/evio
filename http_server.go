// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shabicheng/evio/logger"
)

type HttpContext struct {
	is  InputStream
	req *HttpRequest
}

type HttpRequest struct {
	conn                 Conn
	proto, method        string
	path, query          string
	head, body           string
	remoteAddr           string
	bodyLen              int
	interf               string
	callMethod           string
	parameterTypesString string
	parameter            string
}

var ParseFormBodyError = errors.New("ParseFormBodyError")

func (hq *HttpRequest) ParseFormBody() error {
	kvs := strings.Split(hq.body, "&")
	if len(kvs) != 4 {
		return ParseFormBodyError
	}
	hq.interf = kvs[0][0:strings.IndexByte(kvs[0], '=')]
	hq.callMethod = kvs[1][0:strings.IndexByte(kvs[1], '=')]
	hq.parameterTypesString = kvs[2][0:strings.IndexByte(kvs[2], '=')]
	hq.parameter = kvs[3][0:strings.IndexByte(kvs[3], '=')]
	return nil
}

func (hq *HttpRequest) Response(req *AgentRequest) error {
	out := AppendResp(nil, "200 OK", "", string(req.Param))
	return hq.conn.Send(out)
}

func ServeListenHttp(loops int, port int, workerQueue chan *HttpRequest) error {

	var events Events
	events.NumLoops = loops
	events.Serving = func(srv Server) (action Action) {
		logger.Info("http server started on port %d (loops: %d)", port, srv.NumLoops)
		return
	}

	events.Opened = func(c Conn) (out []byte, opts Options, action Action) {
		c.SetContext(&HttpContext{})

		//logger.Info("opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	}

	events.Closed = func(c Conn, err error) (action Action) {
		//logger.Info("closed: %s: %s", c.LocalAddr().String(), c.RemoteAddr().String())
		return
	}

	events.Data = func(c Conn, in []byte) (out []byte, action Action) {
		//logger.Info("Data: laddr: %v: raddr: %v, data", c.LocalAddr(), c.RemoteAddr(), string(in))
		if in == nil {
			return
		}
		httpContext := c.Context().(*HttpContext)

		if httpContext.req == nil {
			httpContext.req = &HttpRequest{}
			httpContext.is.b = nil
			httpContext.req.conn = c
			// handle the request
			httpContext.req.remoteAddr = c.RemoteAddr().String()
		}
		data := httpContext.is.Begin(in)
		// process the pipeline
		for {
			leftover, err, ready := parsereq(data, httpContext.req)
			//logger.Info("result %v %v %v \n", leftover, err, ready)
			if err != nil {
				logger.Info("bad thing happened\n")
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				break
			} else if !ready {
				// request not ready, yet
				data = leftover
				break
			}
			err = httpContext.req.ParseFormBody()
			if err != nil {
				logger.Info("parse form body error \n")
				out = AppendResp(out, "500 Error", "", err.Error()+"\n")
				action = Close
				break
			}
			//logger.Info("insert \n")
			//AppendRequest(out, httpContext.req)
			workerQueue <- httpContext.req
			httpContext.req = nil
			httpContext.req = &HttpRequest{}
			httpContext.is.b = nil
			httpContext.req.conn = c
			// handle the request
			httpContext.req.remoteAddr = c.RemoteAddr().String()
			data = leftover
		}
		httpContext.is.End(data)
		return
	}

	// We at least want the single http address.
	addrs := []string{fmt.Sprintf("tcp://:%d?reuseport=true", port)}
	// Start serving!
	return Serve(events, addrs...)
}

// AppendRequest handles the incoming request and appends the response to
// the provided bytes, which is then returned to the caller.
func AppendRequest(b []byte, req *HttpRequest) []byte {
	var res string
	return AppendResp(b, "200 OK", "", res)
}

// AppendResp will append a valid http response to the provide bytes.
// The status param should be the code plus text such as "200 OK".
// The head parameter should be a series of lines ending with "\r\n" or empty.
func AppendResp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server:shabicheng\r\n"...)
	b = append(b, "Connection:keep-alive\r\n"...)

	//b = append(b, "Date: "...)
	//b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	if len(head) > 0 {
		b = append(b, head...)
	}
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

// parsereq is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
func parsereq(data []byte, req *HttpRequest) (leftover []byte, err error, ready bool) {
	sdata := string(data)
	if req.bodyLen > 0 {
		if len(sdata) < req.bodyLen {
			return data, nil, false
		}

		req.body = string(data[0:req.bodyLen])
		return data[req.bodyLen:], nil, true
	}
	var i, s int
	var top string
	var clen int
	var q = -1
	// method, path, proto line
	for ; i < len(sdata); i++ {
		if sdata[i] == ' ' {
			req.method = sdata[s:i]
			for i, s = i+1, i+1; i < len(sdata); i++ {
				if sdata[i] == '?' && q == -1 {
					q = i - s
				} else if sdata[i] == ' ' {
					if q != -1 {
						req.path = sdata[s:q]
						req.query = req.path[q+1 : i]
					} else {
						req.path = sdata[s:i]
					}
					for i, s = i+1, i+1; i < len(sdata); i++ {
						if sdata[i] == '\n' && sdata[i-1] == '\r' {
							req.proto = sdata[s:i]
							i, s = i+1, i+1
							break
						}
					}
					break
				}
			}
			break
		}
	}
	if req.proto == "" {
		return data, nil, false
	}
	top = sdata[:s]
	for ; i < len(sdata); i++ {
		if i > 1 && sdata[i] == '\n' && sdata[i-1] == '\r' {
			line := sdata[s : i-1]
			s = i + 1
			if line == "" {
				req.bodyLen = clen
				req.head = sdata[len(top)+2 : i+1]
				i++
				if clen > 0 {
					if len(sdata[i:]) < clen {
						return data[i:], nil, false
					}
					req.body = sdata[i : i+clen]
					i += clen
				}
				return data[i:], nil, true
			}
			if strings.HasPrefix(line, "Content-Length:") {
				n, err := strconv.ParseInt(strings.TrimSpace(line[len("Content-Length:"):]), 10, 64)
				if err == nil {
					clen = int(n)
				}
			}
		}
	}
	// not enough data
	return data, nil, false
}
