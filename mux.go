package zmux

import (
	"fmt"
	"io"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type Mux interface {
	Accept() (*Channel, error)
	AcceptWithSize(bufferSize int) (*Channel, error)
	Open() (*Channel, error)
	OpenWithSize(bufferSize int) (*Channel, error)
}

func New(conn io.ReadWriteCloser) Mux {
	return NewWithConfig(conn, DefaultConfig)
}

func NewWithConfig(conn io.ReadWriteCloser, cfg Config) Mux {
	cfg.normalize()
	m := &mux{
		base:           conn,
		channels:       make(map[uint16]*Channel),
		sb:             NewLimitBuffer(cfg.SendBufferSize),
		notifyOpen:     make(chan struct{}),
		notifyAccepted: make(chan struct{}),
		accepted:       make(chan uint16),
		cfg:            cfg,
		nextId:         math.MaxUint16,
	}
	go m.recv()
	go m.send()
	return m
}

type mux struct {
	base     io.ReadWriteCloser
	channels map[uint16]*Channel
	mu       sync.RWMutex
	sb       *LimitBuffer

	notifyOpen     chan struct{}
	notifyAccepted chan struct{}

	nextId uint32
	cfg    Config

	accepted chan uint16
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
			m.notifyOpen <- struct{}{}
			<-m.notifyAccepted
		case ACCEPTED:
			m.accepted <- h.ConnID()
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

// Принимает канал. Вызывать только на сервере
func (m *mux) Accept() (*Channel, error) {
	return m.AcceptWithSize(m.cfg.RecvBuffersSize)
}

func (m *mux) AcceptWithSize(bufferSize int) (*Channel, error) {
	<-m.notifyOpen

	cid := uint16(atomic.AddUint32(&m.nextId, 1))
	h := NewHeader(ACCEPTED, cid, 0)

	_, err := m.sb.Write(h[:])
	if err != nil {
		m.notifyAccepted <- struct{}{}
		return nil, fmt.Errorf("failed to write frame in buffer: %w", err)
	}

	c := &Channel{
		rb: NewLimitBuffer(bufferSize),
		wb: m.sb,
		id: cid,
	}

	m.mu.Lock()
	m.channels[cid] = c
	m.mu.Unlock()

	m.notifyAccepted <- struct{}{}
	return c, nil
}

// Открывает канал. Вызывать только на клиенте
func (m *mux) Open() (*Channel, error) {
	return m.OpenWithSize(m.cfg.RecvBuffersSize)
}

func (m *mux) OpenWithSize(bufferSize int) (*Channel, error) {
	h := NewHeader(OPEN, 0, 0)
	time.Sleep(time.Second)
	_, err := m.sb.Write(h[:])
	if err != nil {
		return nil, fmt.Errorf("failed to write frame in buffer: %w", err)
	}

	cid := <-m.accepted

	c := &Channel{
		rb: NewLimitBuffer(bufferSize),
		wb: m.sb,
		id: cid,
	}

	m.mu.Lock()
	m.channels[cid] = c
	m.mu.Unlock()

	return c, nil
}
