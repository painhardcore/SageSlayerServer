package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// MaxMessageSize defines the maximum allowed size
const MaxMessageSize = 5 * 1024 * 1024 // 5 mb

// ReadMessage reads a length-prefixed message from the connection.
func ReadMessage(conn net.Conn) ([]byte, error) {
	// Read the length header (4 bytes for uint32)
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, fmt.Errorf("failed to read message length: %v", err)
	}

	// Decode the length
	length := binary.BigEndian.Uint32(lengthBuf)
	if length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum allowed size", length)
	}

	// Read the message data
	msgBuf := make([]byte, length)
	if _, err := io.ReadFull(conn, msgBuf); err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	return msgBuf, nil
}

// WriteMessage writes a length-prefixed message to the connection.
func WriteMessage(conn net.Conn, data []byte) error {
	length := uint32(len(data))
	if length > MaxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum allowed size", length)
	}

	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, length)

	// Write the length header followed by the message data
	if _, err := conn.Write(lengthBuf); err != nil {
		return fmt.Errorf("failed to write message length: %v", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write message data: %v", err)
	}
	return nil
}
