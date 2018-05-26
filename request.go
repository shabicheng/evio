package main

type Request interface {
	Response(req *AgentRequest) error
}
