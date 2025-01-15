package zmux

type Channel struct {
	id uint16
	rb *LimitBuffer
	wb *LimitBuffer
}

func (c *Channel) Read(b []byte) (n int, err error) {
	return c.rb.Read(b)
}

func (c *Channel) Write(b []byte) (n int, err error) {
	h := NewHeader(PAYLOAD, c.id, uint32(len(b)))
	return c.wb.Write(append(h[:], b...))
}
