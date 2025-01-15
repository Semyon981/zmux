package zmux

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

type Mux interface {
	Accept() (*Channel, error)
	Open() (*Channel, error)
}

func NewMux(conn net.Conn) Mux {
	m := &mux{
		base:       conn,
		channels:   make(map[uint16]*Channel),
		sb:         NewLimitBuffer(1024 * 16),
		notifyOpen: make(chan *Channel),
	}
	go m.recv()
	go m.send()
	return m
}

type mux struct {
	base       net.Conn
	channels   map[uint16]*Channel
	mu         sync.RWMutex
	sb         *LimitBuffer
	notifyOpen chan *Channel

	nextId uint32
}

func (m *mux) recv() {
	for {
		var h Header
		_, err := io.ReadFull(m.base, h[:])
		if err != nil {
			log.Printf("failed to read header: %s\n", err)
			return
		}

		switch h.FrameType() {
		case OPEN:
			c := &Channel{
				rb: NewLimitBuffer(1024),
				wb: m.sb,
				id: h.ConnID(),
			}

			m.mu.Lock()
			m.channels[h.ConnID()] = c
			m.mu.Unlock()

			m.notifyOpen <- c
		case PAYLOAD:
			m.mu.RLock()
			c, ok := m.channels[h.ConnID()]
			m.mu.RUnlock()
			if ok {
				_, err := c.rb.ReadFrom(io.LimitReader(m.base, int64(h.PayloadSize())))
				if err != nil {
					log.Printf("failed to write payload in buffer: %s\n", err)
					return
				}
			} else {
				log.Printf("channel is closed: %s\n", err)
				return
			}
		default:
			log.Printf("unsupported frame type: %s\n", err)
			return
		}

	}
}

func (m *mux) send() {
	for {
		_, err := m.sb.WriteTo(m.base)
		if err != nil {
			log.Printf("failed to send data: %s\n", err)
			return
		}
	}
}

func (m *mux) Accept() (*Channel, error) {
	return <-m.notifyOpen, nil
}

func (m *mux) Open() (*Channel, error) {
	cid := uint16(atomic.AddUint32(&m.nextId, 1))
	h := NewHeader(OPEN, cid, 0)

	c := &Channel{
		rb: NewLimitBuffer(1024),
		wb: m.sb,
		id: cid,
	}

	m.mu.Lock()
	m.channels[cid] = c
	m.mu.Unlock()

	_, err := m.sb.Write(h[:])
	if err != nil {
		return nil, fmt.Errorf("failed to write frame in buffer: %w", err)
	}
	return c, nil
}
