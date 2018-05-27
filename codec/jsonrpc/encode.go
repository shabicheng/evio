package jsonrpc

import (
	"bytes"
	"unicode/utf8"
	"reflect"
	"unsafe"
)

type Encoder struct {
	buffer []byte
}

func NewEncoder() *Encoder {
	var buffer = make([]byte, 64)

	return &Encoder{
		buffer: buffer[:0],
	}
}

func (e *Encoder) Buffer() []byte {
	return e.buffer[:]
}

func (e *Encoder) Append(buf []byte) {
	e.buffer = append(e.buffer, buf[:]...)
}

func encByte(b []byte, t ...byte) []byte {
	return append(b, t...)
}

/////////////////////////////////////////
// String
/////////////////////////////////////////

// # UTF-8 encoded character string split into 64k chunks
// ::= x52 b1 b0 <utf8-data> string  # non-final chunk
// ::= 'S' b1 b0 <utf8-data>         # string of length 0-65535
// ::= [x00-x1f] <utf8-data>         # string of length 0-31
// ::= [x30-x34] <utf8-data>         # string of length 0-1023
func (e *Encoder) encString(v string, b []byte) []byte {
	var (
		vBuf = *bytes.NewBufferString(v)
		vLen = utf8.RuneCountInString(v)

		vChunk = func(length int) {
			for i := 0; i < length; i++ {
				if r, s, err := vBuf.ReadRune(); s > 0 && err == nil {
					// b = append(b, []byte(string(r))...)
					b = append(b, _slice(string(r))...) // 直接基于r的内存空间把它转换为[]byte
				}
			}
		}
	)

	if v == "" {
		return encByte(b, BC_STRING_DIRECT)
	}

	for {
		vLen = utf8.RuneCount(vBuf.Bytes())
		if vLen == 0 {
			break
		}
		if vLen > CHUNK_SIZE {
			b = encByte(b, BC_STRING_CHUNK)
			b = encByte(b, PackUint16(uint16(CHUNK_SIZE))...)
			vChunk(CHUNK_SIZE)
		} else {
			if vLen <= int(STRING_DIRECT_MAX) {
				b = encByte(b, byte(vLen+int(BC_STRING_DIRECT)))
			} else if vLen <= int(STRING_SHORT_MAX) {
				b = encByte(b, byte((vLen>>8)+int(BC_STRING_SHORT)), byte(vLen))
			} else {
				b = encByte(b, BC_STRING)
				b = encByte(b, PackUint16(uint16(vLen))...)
			}
			vChunk(vLen)
		}
	}

	return b
}

func _slice(s string) (b []byte) {
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pbytes.Data = pstring.Data
	pbytes.Len = pstring.Len
	pbytes.Cap = pstring.Len
	return
}

/////////////////////////////////////////
// Binary, []byte
/////////////////////////////////////////

// # 8-bit binary data split into 64k chunks
// ::= x41(A) b1 b0 <binary-data> binary # non-final chunk
// ::= x42(B) b1 b0 <binary-data>        # final chunk
// ::= [x20-x2f] <binary-data>           # binary data of length 0-15
// ::= [x34-x37] <binary-data>           # binary data of length 0-1023
func (e *Encoder) encBinary(v []byte, b []byte) []byte {
	var (
		length  uint16
		vLength int
	)

	if len(v) == 0 {
		return encByte(b, BC_NULL)
	}

	vLength = len(v)

	for vLength > 0 {
		if vLength > CHUNK_SIZE {
			length = CHUNK_SIZE
			b = encByte(b, byte(BC_BINARY_CHUNK), byte(length>>8), byte(length))
		} else {
			length = uint16(vLength)

			if vLength <= int(BINARY_DIRECT_MAX) {
				b = encByte(b, byte(int(BC_BINARY_DIRECT)+vLength))
			} else if vLength <= int(BINARY_SHORT_MAX) {
				b = encByte(b, byte(int(BC_BINARY_SHORT)+vLength>>8), byte(vLength))
			} else {
				b = encByte(b, byte(BC_BINARY), byte(vLength>>8), byte(vLength))
			}
		}

		b = append(b, v[:length]...)
		v = v[length:]
		vLength = len(v)
	}

	return b
}
