package kiro

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// Value types for EventStream headers
const (
	ValueTypeBoolTrue    byte = 0
	ValueTypeBoolFalse   byte = 1
	ValueTypeByte        byte = 2
	ValueTypeShort       byte = 3
	ValueTypeInt         byte = 4
	ValueTypeLong        byte = 5
	ValueTypeBytes       byte = 6
	ValueTypeString      byte = 7
	ValueTypeTimestamp   byte = 8
	ValueTypeUUID        byte = 9
)

// Event types
const (
	EventTypeException = "exception"
	EventTypeEvent     = "event"
)

// Message types
const (
	MessageTypeException = "exception"
	MessageTypeEvent     = "event"
)

// HeaderValue represents a parsed header value
type HeaderValue struct {
	Type       byte
	BoolValue  bool
	ByteValue  byte
	ShortValue int16
	IntValue   int32
	LongValue  int64
	BytesValue []byte
	StringValue string
}

// EventStreamMessage represents a parsed EventStream message
type EventStreamMessage struct {
	Headers map[string]HeaderValue
	Payload []byte
}

// GetHeaderString returns a header value as string
func (m *EventStreamMessage) GetHeaderString(name string) string {
	if v, ok := m.Headers[name]; ok {
		return v.StringValue
	}
	return ""
}

// RobustEventStreamParser parses AWS EventStream binary protocol
type RobustEventStreamParser struct {
	buffer     []byte
	errorCount int
}

// NewRobustEventStreamParser creates a new parser
func NewRobustEventStreamParser() *RobustEventStreamParser {
	return &RobustEventStreamParser{}
}

// AddData adds data to the parser buffer
func (p *RobustEventStreamParser) AddData(data []byte) {
	p.buffer = append(p.buffer, data...)
}

// GetMessages parses and returns all complete messages from the buffer
func (p *RobustEventStreamParser) GetMessages() ([]EventStreamMessage, error) {
	var messages []EventStreamMessage

	for len(p.buffer) >= EventStreamMinMessageSize {
		if p.errorCount >= ParserMaxErrors {
			return messages, fmt.Errorf("too many parse errors (%d)", p.errorCount)
		}

		// Read prelude: total_length (4) + header_length (4) + prelude_crc (4)
		if len(p.buffer) < 12 {
			break
		}

		totalLength := binary.BigEndian.Uint32(p.buffer[0:4])
		headerLength := binary.BigEndian.Uint32(p.buffer[4:8])
		preludeCRC := binary.BigEndian.Uint32(p.buffer[8:12])

		// Validate total length
		if totalLength < EventStreamMinMessageSize || totalLength > EventStreamMaxMessageSize {
			p.errorCount++
			p.buffer = p.buffer[1:]
			continue
		}

		// Check if we have the complete message
		if uint32(len(p.buffer)) < totalLength {
			break
		}

		// Validate prelude CRC
		computedPreludeCRC := crc32.ChecksumIEEE(p.buffer[0:8])
		if computedPreludeCRC != preludeCRC {
			p.errorCount++
			p.buffer = p.buffer[1:]
			continue
		}

		// Validate message CRC
		messageCRCOffset := totalLength - 4
		messageCRC := binary.BigEndian.Uint32(p.buffer[messageCRCOffset : messageCRCOffset+4])
		computedMessageCRC := crc32.ChecksumIEEE(p.buffer[0:messageCRCOffset])
		if computedMessageCRC != messageCRC {
			p.errorCount++
			p.buffer = p.buffer[1:]
			continue
		}

		// Parse headers
		headersStart := uint32(12)
		headersEnd := headersStart + headerLength
		headers, err := parseHeaders(p.buffer[headersStart:headersEnd])
		if err != nil {
			p.errorCount++
			p.buffer = p.buffer[totalLength:]
			continue
		}

		// Extract payload
		payloadStart := headersEnd
		payloadEnd := messageCRCOffset
		var payload []byte
		if payloadEnd > payloadStart {
			payload = make([]byte, payloadEnd-payloadStart)
			copy(payload, p.buffer[payloadStart:payloadEnd])
		}

		messages = append(messages, EventStreamMessage{
			Headers: headers,
			Payload: payload,
		})

		p.buffer = p.buffer[totalLength:]
	}

	return messages, nil
}

// parseHeaders parses EventStream headers from binary data
func parseHeaders(data []byte) (map[string]HeaderValue, error) {
	headers := make(map[string]HeaderValue)
	offset := 0

	for offset < len(data) {
		// Read header name length (1 byte)
		if offset >= len(data) {
			return headers, fmt.Errorf("unexpected end of headers at name length")
		}
		nameLength := int(data[offset])
		offset++

		// Read header name
		if offset+nameLength > len(data) {
			return headers, fmt.Errorf("unexpected end of headers at name")
		}
		name := string(data[offset : offset+nameLength])
		offset += nameLength

		// Read value type (1 byte)
		if offset >= len(data) {
			return headers, fmt.Errorf("unexpected end of headers at value type")
		}
		valueType := data[offset]
		offset++

		// Read value based on type
		hv := HeaderValue{Type: valueType}
		switch valueType {
		case ValueTypeBoolTrue:
			hv.BoolValue = true
		case ValueTypeBoolFalse:
			hv.BoolValue = false
		case ValueTypeByte:
			if offset >= len(data) {
				return headers, fmt.Errorf("unexpected end of headers at byte value")
			}
			hv.ByteValue = data[offset]
			offset++
		case ValueTypeShort:
			if offset+2 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at short value")
			}
			hv.ShortValue = int16(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
		case ValueTypeInt:
			if offset+4 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at int value")
			}
			hv.IntValue = int32(binary.BigEndian.Uint32(data[offset : offset+4]))
			offset += 4
		case ValueTypeLong:
			if offset+8 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at long value")
			}
			hv.LongValue = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
			offset += 8
		case ValueTypeBytes:
			if offset+2 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at bytes length")
			}
			bytesLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
			if offset+bytesLen > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at bytes value")
			}
			hv.BytesValue = make([]byte, bytesLen)
			copy(hv.BytesValue, data[offset:offset+bytesLen])
			offset += bytesLen
		case ValueTypeString:
			if offset+2 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at string length")
			}
			strLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
			if offset+strLen > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at string value")
			}
			hv.StringValue = string(data[offset : offset+strLen])
			offset += strLen
		case ValueTypeTimestamp:
			if offset+8 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at timestamp value")
			}
			hv.LongValue = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
			offset += 8
		case ValueTypeUUID:
			if offset+16 > len(data) {
				return headers, fmt.Errorf("unexpected end of headers at uuid value")
			}
			hv.BytesValue = make([]byte, 16)
			copy(hv.BytesValue, data[offset:offset+16])
			offset += 16
		default:
			return headers, fmt.Errorf("unknown header value type: %d", valueType)
		}

		headers[name] = hv
	}

	return headers, nil
}
