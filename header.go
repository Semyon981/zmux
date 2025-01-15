package zmux

import "encoding/binary"

type FrameType uint8

const (
	PAYLOAD FrameType = iota
	OPEN
)

type Header [7]byte

func NewHeader(ft FrameType, id uint16, sz uint32) Header {
	var h Header
	h.SetFrameType(ft)
	h.SetConnID(id)
	h.SetPayloadSize(sz)
	return h
}

func (h *Header) FrameType() FrameType {
	return FrameType(h[0])
}

func (h *Header) ConnID() uint16 {
	return binary.BigEndian.Uint16(h[1:3])
}

func (h *Header) PayloadSize() uint32 {
	return binary.BigEndian.Uint32(h[3:])
}

func (h *Header) SetFrameType(ft FrameType) {
	h[0] = byte(ft)
}

func (h *Header) SetConnID(id uint16) {
	binary.BigEndian.PutUint16(h[1:3], id)
}

func (h *Header) SetPayloadSize(sz uint32) {
	binary.BigEndian.PutUint32(h[3:], sz)
}
