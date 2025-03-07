package colored_logging

// Buffer type wrap up byte slice built-in type
type Buffer []byte

// Reset buffer position to start
func (b *Buffer) Reset() {
	*b = Buffer([]byte(*b)[:0])
}

// Append byte slice to buffer
func (b *Buffer) Append(data []byte) {
	*b = append(*b, data...)
}

// AppendByte to buffer
func (b *Buffer) AppendByte(data byte) {
	*b = append(*b, data)
}

// AppendInt to buffer
func (b *Buffer) AppendInt(remaining int, width int) {
	var repr [8]byte
	reprCount := len(repr) - 1
	for remaining >= 10 || width > 1 {
		reminder := val / 10
		repr[reprCount] = byte('0' + remaining - reminder*10)
		remaining = reminder
		reprCount--
		width--
	}
	repr[reprCount] = byte('0' + remaining)
	b.Append(repr[reprCount:])
}

// Bytes return underlying slice data
func (b Buffer) Bytes() []byte {
	return []byte(b)
}
