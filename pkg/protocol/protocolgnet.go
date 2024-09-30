package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/panjf2000/gnet/v2"
)

func ReadMessageGnet(c gnet.Conn) ([]byte, error) {
	// Check if we have at least 4 bytes for the length header
	if c.InboundBuffered() < 4 {
		return nil, nil // Not enough data yet
	}

	// Peek at the length header
	lengthBuf, _ := c.Peek(4)
	length := binary.BigEndian.Uint32(lengthBuf)

	// Check if the message size exceeds the maximum allowed size
	if length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum allowed size", length)
	}

	// Check if we have the full message (length + data)
	if c.InboundBuffered() < int(4+length) {
		return nil, nil // Not enough data yet
	}

	// Now retrieve the full message
	_, _ = c.Discard(4) // Discard the length header
	message, _ := c.Peek(int(length))

	// Discard the message bytes from the buffer
	_, _ = c.Discard(int(length))

	return message, nil
}

func WriteMessageGnet(c gnet.Conn, data []byte) error {
	length := uint32(len(data))
	if length > MaxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum allowed size", length)
	}

	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, length)

	// Write the length header followed by the message data asynchronously
	return c.AsyncWrite(append(lengthBuf, data...), nil)
}
