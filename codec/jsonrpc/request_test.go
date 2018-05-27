package jsonrpc

import (
	"testing"
	"mesh-agent.go/codec"
)

func TestEncodeRequest(t *testing.T) {
	attachments := make(map[string]string)
	attachments["path"] = "/cat/r/test"
	data := make([]byte, 0, 10)

	message := &codec.Message{
		ID:          123,
		Version:     "1.0",
		Type:        1,
		ServicePath: "/a/b/c",
		Interface:   "wyp", // Service
		Method:      "abc",
		Data: &codec.RpcInvocation{
			Method:         "test.a",
			ParameterTypes: "java.lang.string",
			Args:           data,
			Attachments:    attachments,
		},
	}

	data, e := packRequest(message)

	if e != nil {
		// do something
	}

	unpackResponse(data, &Response{})
}
