package jsonrpc

import (
	"bufio"
	"bytes"
)

type Decoder struct {
	reader *bufio.Reader
}

func NewDecoder(b []byte) *Decoder {
	return &Decoder{reader: bufio.NewReader(bytes.NewReader(b))}
}

// 读取当前字节,指针不前移
func (d *Decoder) peekByte() byte {
	return d.peek(1)[0]
}

// 获取缓冲长度
func (d *Decoder) len() int {
	d.peek(1) //需要先读一下资源才能得到已缓冲的长度
	return d.reader.Buffered()
}

// 读取 Decoder 结构中的一个字节,并后移一个字节
func (d *Decoder) readByte() (byte, error) {
	return d.reader.ReadByte()
}

// 前移一个字节
func (d *Decoder) unreadByte() error {
	return d.reader.UnreadByte()
}

// 读取指定长度的字节,并后移len(b)个字节
func (d *Decoder) next(b []byte) (int, error) {
	return d.reader.Read(b)
}

// 读取指定长度字节,指针不后移
func (d *Decoder) peek(n int) []byte {
	b, _ := d.reader.Peek(n)
	return b
}

// 读取len(s)的 utf8 字符
func (d *Decoder) nextRune(s []rune) []rune {
	var (
		n  int
		i  int
		r  rune
		ri int
		e  error
	)

	n = len(s)
	s = s[:0]
	for i = 0; i < n; i++ {
		if r, ri, e = d.reader.ReadRune(); e == nil && ri > 0 {
			s = append(s, r)
		}
	}

	return s
}
