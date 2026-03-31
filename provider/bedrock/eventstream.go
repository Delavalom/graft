package bedrock

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

const (
	preludeSize          = 12 // 4 (total length) + 4 (headers length) + 4 (prelude CRC)
	crcSize              = 4
	headerValueTypeString = 7
)

// eventStreamDecoder reads AWS Event Stream binary frames from a reader.
type eventStreamDecoder struct {
	reader io.Reader
}

// newEventStreamDecoder creates a new event stream decoder.
func newEventStreamDecoder(r io.Reader) *eventStreamDecoder {
	return &eventStreamDecoder{reader: r}
}

// readEvent reads the next frame from the event stream.
// Returns io.EOF when there are no more frames.
func (d *eventStreamDecoder) readEvent() (eventType string, payload []byte, err error) {
	// Read prelude: total length (4) + headers length (4) + prelude CRC (4).
	prelude := make([]byte, preludeSize)
	if _, err := io.ReadFull(d.reader, prelude); err != nil {
		return "", nil, err
	}

	totalLength := binary.BigEndian.Uint32(prelude[0:4])
	headersLength := binary.BigEndian.Uint32(prelude[4:8])
	preludeCRC := binary.BigEndian.Uint32(prelude[8:12])

	// Verify prelude CRC (covers first 8 bytes).
	computedPreludeCRC := crc32.Checksum(prelude[0:8], crc32cTable)
	if computedPreludeCRC != preludeCRC {
		return "", nil, fmt.Errorf("eventstream: prelude CRC mismatch: got %08x, want %08x", computedPreludeCRC, preludeCRC)
	}

	// Read the rest of the frame.
	remaining := int(totalLength) - preludeSize
	rest := make([]byte, remaining)
	if _, err := io.ReadFull(d.reader, rest); err != nil {
		return "", nil, fmt.Errorf("eventstream: incomplete frame: %w", err)
	}

	// Verify message CRC (covers entire frame minus last 4 bytes).
	messageCRC := binary.BigEndian.Uint32(rest[remaining-crcSize:])
	frame := make([]byte, int(totalLength)-crcSize)
	copy(frame, prelude)
	copy(frame[preludeSize:], rest[:remaining-crcSize])
	computedMessageCRC := crc32.Checksum(frame, crc32cTable)
	if computedMessageCRC != messageCRC {
		return "", nil, fmt.Errorf("eventstream: message CRC mismatch: got %08x, want %08x", computedMessageCRC, messageCRC)
	}

	// Parse headers.
	headersData := rest[:headersLength]
	headers := parseHeaders(headersData)
	eventType = headers[":event-type"]

	// Extract payload.
	payloadLength := int(totalLength) - preludeSize - int(headersLength) - crcSize
	payload = rest[headersLength : headersLength+uint32(payloadLength)]

	return eventType, payload, nil
}

// parseHeaders parses binary event stream headers.
// Only handles string type (type 7).
func parseHeaders(data []byte) map[string]string {
	headers := make(map[string]string)
	offset := 0
	for offset < len(data) {
		if offset >= len(data) {
			break
		}
		// Name length (1 byte).
		nameLen := int(data[offset])
		offset++
		// Name.
		name := string(data[offset : offset+nameLen])
		offset += nameLen
		// Value type (1 byte).
		valueType := data[offset]
		offset++
		if valueType == headerValueTypeString {
			// Value length (2 bytes, big-endian).
			valueLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
			// Value.
			value := string(data[offset : offset+valueLen])
			offset += valueLen
			headers[name] = value
		}
	}
	return headers
}

// encodeHeader encodes a single string header in event stream binary format.
func encodeHeader(name, value string) []byte {
	buf := make([]byte, 0, 1+len(name)+1+2+len(value))
	buf = append(buf, byte(len(name)))
	buf = append(buf, []byte(name)...)
	buf = append(buf, headerValueTypeString)
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(value)))
	buf = append(buf, []byte(value)...)
	return buf
}

// encodeEventStreamFrame builds a valid event stream frame with the given
// event type and payload. It includes three headers: :event-type,
// :content-type ("application/json"), and :message-type ("event").
func encodeEventStreamFrame(eventType string, payload []byte) []byte {
	// Build headers.
	var headers []byte
	headers = append(headers, encodeHeader(":event-type", eventType)...)
	headers = append(headers, encodeHeader(":content-type", "application/json")...)
	headers = append(headers, encodeHeader(":message-type", "event")...)

	headersLength := len(headers)
	totalLength := preludeSize + headersLength + len(payload) + crcSize

	// Build the frame.
	frame := make([]byte, 0, totalLength)

	// Prelude: total length + headers length.
	frame = binary.BigEndian.AppendUint32(frame, uint32(totalLength))
	frame = binary.BigEndian.AppendUint32(frame, uint32(headersLength))

	// Prelude CRC (over first 8 bytes).
	preludeCRC := crc32.Checksum(frame[:8], crc32cTable)
	frame = binary.BigEndian.AppendUint32(frame, preludeCRC)

	// Headers and payload.
	frame = append(frame, headers...)
	frame = append(frame, payload...)

	// Message CRC (over entire frame so far).
	messageCRC := crc32.Checksum(frame, crc32cTable)
	frame = binary.BigEndian.AppendUint32(frame, messageCRC)

	return frame
}
