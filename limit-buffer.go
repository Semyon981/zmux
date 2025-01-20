package zmux

import (
	"io"
	"sync"
)

func NewLimitBuffer(size int) *LimitBuffer {
	return &LimitBuffer{
		buf:  make([]byte, size),
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

type LimitBuffer struct {
	buf   []byte
	cond  *sync.Cond
	start int
	sz    int
	mu    sync.Mutex
}

func (b *LimitBuffer) Read(p []byte) (int, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	for b.sz == 0 {
		b.cond.Wait()
	}

	n := min(len(p), b.sz)
	end := b.start + n
	if end <= len(b.buf) {
		copy(p, b.buf[b.start:end])
	} else {
		copy(p, b.buf[b.start:])
		copy(p[len(b.buf)-b.start:], b.buf[:end-len(b.buf)])
	}

	b.start = (b.start + n) % len(b.buf)
	b.sz -= n

	b.cond.Broadcast()
	return n, nil
}

func (b *LimitBuffer) WriteTo(w io.Writer) (int64, error) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	for b.sz == 0 {
		b.cond.Wait()
	}

	end := b.start + b.sz
	var writed int64
	var err error
	var n int
	if end <= len(b.buf) {
		n, err = w.Write(b.buf[b.start:end])
		writed, b.sz = writed+int64(n), b.sz-n
	} else {
		n, err := w.Write(b.buf[b.start:])
		writed, b.sz = writed+int64(n), b.sz-n
		if err == nil && b.start != 0 {
			n, err = w.Write(b.buf[:end%len(b.buf)])
			writed, b.sz = writed+int64(n), b.sz-n
		}
	}

	b.cond.Broadcast()
	return writed, err
}

func (b *LimitBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	n := 0
	for n != len(p) {
		for len(b.buf) == b.sz {
			b.cond.Wait()
		}

		toWrite := min(len(b.buf)-b.sz, len(p)-n)
		writePos := (b.start + b.sz) % len(b.buf)

		if writePos+toWrite <= len(b.buf) {
			copy(b.buf[writePos:], p[n:n+toWrite])
		} else {
			fpidx := n + len(b.buf) - writePos
			copy(b.buf[writePos:], p[n:fpidx])
			copy(b.buf, p[fpidx:n+toWrite])
		}

		b.sz += toWrite
		n += toWrite

		b.cond.Broadcast()
	}
	return n, nil
}

func (b *LimitBuffer) ReadFrom(r io.Reader) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cond.L.Lock()
	defer b.cond.L.Unlock()

	var readed int64
	var err error
	var n int
	for err == nil {
		for len(b.buf) == b.sz {
			b.cond.Wait()
		}
		free := len(b.buf) - b.sz
		freeStart := (b.start + b.sz) % len(b.buf)
		freeEnd := freeStart + free

		n, err = io.ReadFull(r, b.buf[freeStart:])
		readed, b.sz = readed+int64(n), b.sz+n
		if err == nil && freeEnd > len(b.buf) {
			n, err = io.ReadFull(r, b.buf[:freeEnd%len(b.buf)])
			readed, b.sz = readed+int64(n), b.sz+n
		}
		b.cond.Broadcast()
	}

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	return readed, err
}
