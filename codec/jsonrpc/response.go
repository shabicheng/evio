package jsonrpc

import (
	"encoding/binary"
)

import (
	"mesh-agent.go/codec"
)

const (
	Response_OK                byte = 20
	Response_CLIENT_TIMEOUT    byte = 30
	Response_SERVER_TIMEOUT    byte = 31
	Response_BAD_REQUEST       byte = 40
	Response_BAD_RESPONSE      byte = 50
	Response_SERVICE_NOT_FOUND byte = 60
	Response_SERVICE_ERROR     byte = 70
	Response_SERVER_ERROR      byte = 80
	Response_CLIENT_ERROR      byte = 90

	RESPONSE_WITH_EXCEPTION int32 = 0
	RESPONSE_VALUE          int32 = 1
	RESPONSE_NULL_VALUE     int32 = 2
)

type Response struct {
	ID   int64
	data []byte
}

func unpackResponse(buf []byte, res *Response) error {
	readable := len(buf)

	if readable < HEADER_LENGTH {
		return codec.ErrHeaderNotEnough
	}

	var err error

	header := make([]byte, 0, HEADER_LENGTH)
	header = buf[0:16]

	dataLen := header[12:16]
	dLen := int32(binary.BigEndian.Uint32(dataLen))
	tt := dLen + HEADER_LENGTH

	if int32(readable) < tt {
		return codec.ErrHeaderNotEnough
	}

	data := buf[16:]
	subArray := data[HEADER_LENGTH+2 : len(data)-1]

	res.ID = int64(binary.BigEndian.Uint64(buf[4:12]))
	res.data = subArray

	return err
}
