package media

import (
	"bytes"
	"errors"
	"fmt"
)

var (
	ErrWebMInvalid     = errors.New("invalid WebM stream")
	ErrWebMUnsupported = errors.New("unsupported WebM codec or tracks")
	ErrWebMTooLarge    = errors.New("WebM input exceeds limit")
)

type WebMRecord struct {
	Data         []byte
	Timestamp    int64
	RandomAccess bool
}

type WebMParser struct {
	maxPending int
	pending    []byte
	init       []byte
	validated  bool
}

func NewWebMParser(maxPending int) *WebMParser {
	if maxPending <= 0 {
		maxPending = 4 << 20
	}
	return &WebMParser{maxPending: maxPending}
}

func (p *WebMParser) Init() []byte {
	return cloneBytes(p.init)
}

func (p *WebMParser) Reset() {
	zero(p.pending)
	zero(p.init)
	p.pending = nil
	p.init = nil
	p.validated = false
}

func (p *WebMParser) Close() {
	p.Reset()
}

func (p *WebMParser) Push(data []byte) ([]WebMRecord, error) {
	if len(p.pending)+len(data) > p.maxPending {
		p.Reset()
		return nil, ErrWebMTooLarge
	}
	p.pending = append(p.pending, data...)
	var records []WebMRecord
	for {
		index := bytes.Index(p.pending, clusterID)
		if index < 0 {
			if len(p.pending) > p.maxPending {
				p.Reset()
				return nil, ErrWebMTooLarge
			}
			return records, nil
		}
		if !p.validated {
			if index == 0 || !bytes.HasPrefix(p.pending, ebmlID) {
				p.Reset()
				return nil, ErrWebMInvalid
			}
			init := p.pending[:index]
			if err := validateWebMInit(init); err != nil {
				p.Reset()
				return nil, err
			}
			p.init = cloneBytes(init)
			p.validated = true
			p.pending = p.pending[index:]
		}
		size, sizeLen, unknown, ok := readVINT(p.pending[4:])
		if !ok {
			return records, nil
		}
		if !unknown && size > uint64(p.maxPending) {
			p.Reset()
			return nil, ErrWebMTooLarge
		}
		total := 4 + sizeLen + int(size)
		if unknown {
			next := bytes.Index(p.pending[4+sizeLen:], clusterID)
			if next < 0 {
				return records, nil
			}
			total = 4 + sizeLen + next
		}
		if total < 4+sizeLen || total > p.maxPending {
			p.Reset()
			return nil, ErrWebMTooLarge
		}
		if len(p.pending) < total {
			return records, nil
		}
		raw := cloneBytes(p.pending[:total])
		timestamp, randomAccess, err := parseCluster(raw[4+sizeLen:])
		if err != nil {
			p.Reset()
			return nil, err
		}
		records = append(records, WebMRecord{Data: raw, Timestamp: timestamp, RandomAccess: randomAccess})
		p.pending = append([]byte(nil), p.pending[total:]...)
		if len(p.pending) == 0 {
			return records, nil
		}
	}
}

var (
	ebmlID    = []byte{0x1a, 0x45, 0xdf, 0xa3}
	clusterID = []byte{0x1f, 0x43, 0xb6, 0x75}
)

func validateWebMInit(init []byte) error {
	if !bytes.HasPrefix(init, ebmlID) {
		return ErrWebMInvalid
	}
	tracks := 0
	videoTracks := 0
	for offset := 0; offset+2 <= len(init); {
		index := bytes.IndexByte(init[offset:], 0xae)
		if index < 0 {
			break
		}
		index += offset
		size, sizeLen, _, ok := readVINT(init[index+1:])
		start := index + 1 + sizeLen
		end := start + int(size)
		if ok && size <= uint64(len(init)) && end >= start && end <= len(init) {
			trackType, codec, valid := parseTrackEntry(init[start:end])
			if valid {
				tracks++
				switch trackType {
				case 1:
					videoTracks++
					if codec != "V_VP8" {
						return ErrWebMUnsupported
					}
				case 2:
					return ErrWebMUnsupported
				default:
					return ErrWebMUnsupported
				}
			}
		}
		offset = index + 1
	}
	if tracks != 1 || videoTracks != 1 {
		return ErrWebMUnsupported
	}
	return nil
}

func parseTrackEntry(payload []byte) (uint64, string, bool) {
	var trackType uint64
	var codec string
	for offset := 0; offset < len(payload); {
		id, idLen, ok := readElementID(payload[offset:])
		if !ok {
			return 0, "", false
		}
		size, sizeLen, unknown, ok := readVINT(payload[offset+idLen:])
		if !ok || unknown || size > uint64(len(payload)) {
			return 0, "", false
		}
		start := offset + idLen + sizeLen
		end := start + int(size)
		if end < start || end > len(payload) {
			return 0, "", false
		}
		switch id {
		case 0x83:
			if size != 1 {
				return 0, "", false
			}
			trackType = unsigned(payload[start:end])
		case 0x86:
			codec = string(payload[start:end])
		}
		offset = end
	}
	return trackType, codec, trackType != 0 && codec != ""
}

func parseCluster(payload []byte) (int64, bool, error) {
	var timestamp int64 = -1
	randomAccess := false
	for offset := 0; offset < len(payload); {
		id, idLen, ok := readElementID(payload[offset:])
		if !ok {
			return 0, false, ErrWebMInvalid
		}
		size, sizeLen, unknown, ok := readVINT(payload[offset+idLen:])
		if !ok || unknown {
			return 0, false, ErrWebMInvalid
		}
		start := offset + idLen + sizeLen
		end := start + int(size)
		if end < start || end > len(payload) {
			return 0, false, ErrWebMInvalid
		}
		value := payload[start:end]
		switch id {
		case 0xe7:
			if len(value) == 0 || len(value) > 8 {
				return 0, false, ErrWebMInvalid
			}
			timestamp = int64(unsigned(value))
		case 0xa3:
			key, err := simpleBlockKeyframe(value)
			if err != nil {
				return 0, false, err
			}
			randomAccess = randomAccess || key
		}
		offset = end
	}
	if timestamp < 0 {
		return 0, false, fmt.Errorf("%w: cluster timestamp missing", ErrWebMInvalid)
	}
	return timestamp, randomAccess, nil
}

func simpleBlockKeyframe(block []byte) (bool, error) {
	_, trackLen, _, ok := readVINT(block)
	if !ok || len(block) < trackLen+4 {
		return false, ErrWebMInvalid
	}
	flags := block[trackLen+2]
	if flags&0x06 != 0 {
		return false, ErrWebMUnsupported
	}
	frame := block[trackLen+3:]
	if len(frame) == 0 {
		return false, ErrWebMInvalid
	}
	return flags&0x80 != 0 && frame[0]&0x01 == 0, nil
}

func readElementID(data []byte) (uint64, int, bool) {
	if len(data) == 0 {
		return 0, 0, false
	}
	length := vintLength(data[0])
	if length == 0 || length > 4 || len(data) < length {
		return 0, 0, false
	}
	return unsigned(data[:length]), length, true
}

func readVINT(data []byte) (uint64, int, bool, bool) {
	if len(data) == 0 {
		return 0, 0, false, false
	}
	length := vintLength(data[0])
	if length == 0 || length > 8 || len(data) < length {
		return 0, 0, false, false
	}
	marker := byte(1 << (8 - length))
	value := uint64(data[0] & (marker - 1))
	unknown := data[0]&(marker-1) == marker-1
	for i := 1; i < length; i++ {
		value = value<<8 | uint64(data[i])
		unknown = unknown && data[i] == 0xff
	}
	return value, length, unknown, true
}

func vintLength(first byte) int {
	for i := 0; i < 8; i++ {
		if first&(0x80>>i) != 0 {
			return i + 1
		}
	}
	return 0
}

func unsigned(data []byte) uint64 {
	var value uint64
	for _, b := range data {
		value = value<<8 | uint64(b)
	}
	return value
}
