package jsonrpc

const (
	TAG_READ        = int32(-1)
	ASCII_GAP       = 32
	CHUNK_SIZE      = 4096
	BC_BINARY       = byte('B') // final chunk
	BC_BINARY_CHUNK = byte('A') // non-final chunk

	BC_BINARY_DIRECT  = byte(0x20) // 1-byte length binary
	BINARY_DIRECT_MAX = byte(0x0f)
	BC_BINARY_SHORT   = byte(0x34) // 2-byte length binary
	BINARY_SHORT_MAX  = 0x3ff      // 0-1023 binary

	BC_DATE        = byte(0x4a) // 64-bit millisecond UTC date
	BC_DATE_MINUTE = byte(0x4b) // 32-bit minute UTC date

	BC_DOUBLE = byte('D') // IEEE 64-bit double

	BC_DOUBLE_ZERO  = byte(0x5b)
	BC_DOUBLE_ONE   = byte(0x5c)
	BC_DOUBLE_BYTE  = byte(0x5d)
	BC_DOUBLE_SHORT = byte(0x5e)
	BC_DOUBLE_MILL  = byte(0x5f)

	BC_FALSE = byte('F') // boolean false

	BC_INT = byte('I') // 32-bit int

	INT_DIRECT_MIN = -0x10
	INT_DIRECT_MAX = byte(0x2f)
	BC_INT_ZERO    = byte(0x90)

	INT_BYTE_MIN     = -0x800
	INT_BYTE_MAX     = 0x7ff
	BC_INT_BYTE_ZERO = byte(0xc8)

	BC_END = byte('Z')

	INT_SHORT_MIN     = -0x40000
	INT_SHORT_MAX     = 0x3ffff
	BC_INT_SHORT_ZERO = byte(0xd4)

	BC_LIST_VARIABLE         = byte(0x55)
	BC_LIST_FIXED            = byte('V')
	BC_LIST_VARIABLE_UNTYPED = byte(0x57)
	BC_LIST_FIXED_UNTYPED    = byte(0x58)

	BC_LIST_DIRECT         = byte(0x70)
	BC_LIST_DIRECT_UNTYPED = byte(0x78)
	LIST_DIRECT_MAX        = byte(0x7)

	BC_LONG         = byte('L') // 64-bit signed integer
	LONG_DIRECT_MIN = -0x08
	LONG_DIRECT_MAX = byte(0x0f)
	BC_LONG_ZERO    = byte(0xe0)

	LONG_BYTE_MIN     = -0x800
	LONG_BYTE_MAX     = 0x7ff
	BC_LONG_BYTE_ZERO = byte(0xf8)

	LONG_SHORT_MIN     = -0x40000
	LONG_SHORT_MAX     = 0x3ffff
	BC_LONG_SHORT_ZERO = byte(0x3c)

	BC_LONG_INT = byte(0x59)

	BC_MAP         = byte('M')
	BC_MAP_UNTYPED = byte('H')

	BC_NULL = byte('N') // x4e

	BC_STRING       = byte('S') // final string
	BC_STRING_CHUNK = byte('R') // non-final string

	BC_STRING_DIRECT  = byte(0x00)
	STRING_DIRECT_MAX = byte(0x1f)
	BC_STRING_SHORT   = byte(0x30)
	STRING_SHORT_MAX  = 0x3ff

	BC_TRUE = byte('T')


)
