package protocol_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/painhardcore/SageSlayerServer/pkg/protocol"
)

// MockConn is a mock implementation of net.Conn to simulate network connections
type MockConn struct {
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
}

func NewMockConn() *MockConn {
	return &MockConn{
		readBuffer:  &bytes.Buffer{},
		writeBuffer: &bytes.Buffer{},
	}
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *MockConn) Close() error { return nil }
func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestReadMessage(t *testing.T) {
	mockConn := NewMockConn()

	// Test case: Valid message
	data := []byte("test message")
	writeMessageToBuffer(mockConn.readBuffer, data)

	msg, err := protocol.ReadMessage(mockConn)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if string(msg) != "test message" {
		t.Fatalf("expected message 'test message', got: %s", string(msg))
	}

	// Test case: Message exceeding MaxMessageSize
	mockConn.readBuffer.Reset()
	largeData := make([]byte, protocol.MaxMessageSize+1)
	writeMessageToBuffer(mockConn.readBuffer, largeData)

	_, err = protocol.ReadMessage(mockConn)
	if err == nil {
		t.Fatal("expected error for message exceeding MaxMessageSize")
	}

	// Test case: Failed to read message length
	mockConn.readBuffer.Reset()
	mockConn.readBuffer.Write([]byte{0x01, 0x02}) // Incomplete length header

	_, err = protocol.ReadMessage(mockConn)
	if err == nil {
		t.Fatal("expected error for incomplete length header")
	}
}

func TestWriteMessage(t *testing.T) {
	mockConn := NewMockConn()

	// Test case: Valid message
	data := []byte("test message")
	err := protocol.WriteMessage(mockConn, data)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that the message was written correctly
	expectedLength := uint32(len(data))
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, expectedLength)
	expectedOutput := append(lengthBuf, data...)

	if !bytes.Equal(mockConn.writeBuffer.Bytes(), expectedOutput) {
		t.Fatalf("expected written message to be %x, got %x", expectedOutput, mockConn.writeBuffer.Bytes())
	}

	// Test case: Message exceeding MaxMessageSize
	largeData := make([]byte, protocol.MaxMessageSize+1)
	err = protocol.WriteMessage(mockConn, largeData)
	if err == nil {
		t.Fatal("expected error for message exceeding MaxMessageSize")
	}

	// Test case: Failed to write length
	mockConn.writeBuffer.Reset()
	errConn := NewErrorConn() // Simulate failure in Write
	err = protocol.WriteMessage(errConn, data)
	if err == nil {
		t.Fatal("expected error for failed write")
	}
}

// Helper function to write a length-prefixed message to a buffer (mocking a network write)
func writeMessageToBuffer(buf *bytes.Buffer, data []byte) {
	length := uint32(len(data))
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, length)
	buf.Write(lengthBuf)
	buf.Write(data)
}

// ErrorConn is a mock connection that simulates a write failure
type ErrorConn struct {
	MockConn
}

func NewErrorConn() *ErrorConn {
	return &ErrorConn{}
}

func (e *ErrorConn) Write(b []byte) (int, error) {
	return 0, fmt.Errorf("failed to write")
}
