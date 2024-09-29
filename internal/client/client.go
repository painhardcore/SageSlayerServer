package client

import (
	"fmt"
	"net"
	"time"

	"github.com/painhardcore/SageSlayerServer/pkg/network"
	"github.com/painhardcore/SageSlayerServer/pkg/pow"
	"github.com/painhardcore/SageSlayerServer/pkg/protocol"
	"google.golang.org/protobuf/proto"
)

// Client represents a TCP client.
type Client struct {
	ServerAddr string
}

// NewClient creates a new Client instance.
func NewClient(serverAddr string) *Client {
	return &Client{
		ServerAddr: serverAddr,
	}
}

// RequestQuote connects to the server and requests a quote.
func (c *Client) RequestQuote() error {
	conn, err := net.Dial("tcp", c.ServerAddr)
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}
	defer conn.Close()

	// Set read/write deadlines
	// 30s should be enough
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Receive the challenge
	data, err := protocol.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading challenge: %v", err)
	}

	message := &network.Message{}
	if err := proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("error unmarshalling message: %v", err)
	}

	switch message.Type {
	case network.MessageType_CHALLENGE:
	case network.MessageType_ERROR:
		errorProto := &network.Error{}
		if err := proto.Unmarshal(message.Payload, errorProto); err != nil {
			return fmt.Errorf("error unmarshalling error: %v", err)
		}
		return fmt.Errorf("error receiving the challenge: %s", errorProto)
	default:
		return fmt.Errorf("invalid message type received")
	}

	// Unmarshal challenge
	challengeProto := &network.Challenge{}
	if err := proto.Unmarshal(message.Payload, challengeProto); err != nil {
		return fmt.Errorf("error unmarshalling challenge: %v", err)
	}

	// Step 2: Solve challenge
	nonce, err := pow.SolveChallenge(challengeProto)
	if err != nil {
		return fmt.Errorf("error solving challenge: %v", err)
	}

	// Prepare solution message
	solutionProto := &network.Solution{Nonce: nonce}
	solutionPayload, err := proto.Marshal(solutionProto)
	if err != nil {
		return fmt.Errorf("error marshalling solution: %v", err)
	}

	solutionMessage := &network.Message{
		Type:    network.MessageType_SOLUTION,
		Payload: solutionPayload,
	}
	data, err = proto.Marshal(solutionMessage)
	if err != nil {
		return fmt.Errorf("error marshalling message: %v", err)
	}
	// Send the solution
	if err := protocol.WriteMessage(conn, data); err != nil {
		return fmt.Errorf("error sending solution: %v", err)
	}
	// Get the response
	data, err = protocol.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	message = &network.Message{}
	if err := proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("error unmarshalling message: %v", err)
	}

	switch message.Type {
	case network.MessageType_QUOTE:
		quoteProto := &network.Quote{}
		if err := proto.Unmarshal(message.Payload, quoteProto); err != nil {
			return fmt.Errorf("error unmarshalling quote: %v", err)
		}
		fmt.Println("Quote of the Day:", quoteProto.Text)
	case network.MessageType_ERROR:
		errorProto := &network.Error{}
		if err := proto.Unmarshal(message.Payload, errorProto); err != nil {
			return fmt.Errorf("error unmarshalling error message: %v", err)
		}
		fmt.Println("Error from server:", errorProto.Message)
	default:
		fmt.Println("Unexpected message type received")
	}

	return nil
}
