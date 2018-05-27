package jsonrpc

import (
	"encoding/binary"
)

import (
	"bytes"
	"encoding/json"

	"github.com/shabicheng/evio/codec"
)

/**
 * 协议头是16字节的定长数据
 * 2字节magic字符串0xdabb,0-7高位，8-15低位
 * 1字节的消息标志位。16-20序列id,21 event,22 two way,23请求或响应标识
 * 1字节状态。当消息类型为响应时，设置响应状态。24-31位。
 * 8字节，消息ID,long类型，32-95位。
 * 4字节，消息长度，96-127位
 **/
const (
	// header length.
	HEADER_LENGTH = 16

	// magic header
	MAGIC      = uint16(0xdabb)
	MAGIC_HIGH = byte(0xda)
	MAGIC_LOW  = byte(0xbb)

	// message flag.
	FLAG_REQUEST = byte(0x80)
	FLAG_TWOWAY  = byte(0x40)
	FLAG_EVENT   = byte(0x20) // for heartbeat

	SERIALIZATION_MASK = 0x1f

	DUBBO_VERSION = "2.5.4"
)

func PackRequest(m *codec.Message) ([]byte, error) {
	var (
		byteArray   []byte
		encoder     *Encoder
		args        []byte
		attachments map[string]string
	)

	dubboHeader := [HEADER_LENGTH]byte{MAGIC_HIGH, MAGIC_LOW, FLAG_REQUEST | FLAG_TWOWAY}

	// magic
	byteArray = append(byteArray, dubboHeader[:]...)

	// serialization id, two way flag, event, request/response flag
	byteArray[2] |= byte(m.ID | 6)

	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(m.ID))

	if m.Type == codec.Heartbeat {
		byteArray[2] |= byte(FLAG_EVENT)
	} else {
		byteArray[2] |= byte(FLAG_REQUEST)
	}

	encoder = NewEncoder()

	var jsonBuf bytes.Buffer
	writeJsonObject(&jsonBuf, toJson(DUBBO_VERSION))
	writeJsonObject(&jsonBuf, toJson(m.Interface))
	writeJsonObject(&jsonBuf, toJson(m.Version))
	writeJsonObject(&jsonBuf, toJson(m.Method))

	rpcInvocation, ok := m.Data.(*codec.RpcInvocation)

	if ok {
		args = rpcInvocation.Args
		attachments = rpcInvocation.Attachments

		writeJsonObject(&jsonBuf, toJson(string(args)))
		writeJsonObject(&jsonBuf, toJson(attachments))
	}

	binary.BigEndian.PutUint32(byteArray[12:], uint32(len(jsonBuf.Bytes())))

	encoder.Append(byteArray[:HEADER_LENGTH])
	encoder.Append(encoder.encString(jsonBuf.String(), encoder.buffer))

	return encoder.buffer, nil
}

func toJson(o interface{}) string {
	b, er := json.Marshal(o)

	if er != nil {
		return ""
	}

	return string(b)
}

func writeJsonObject(jsonBuf *bytes.Buffer, json string) {
	jsonBuf.WriteString(json)
	jsonBuf.WriteString("\n")
}
