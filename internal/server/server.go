package server

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/painhardcore/SageSlayerServer/internal/quotes"
	"github.com/painhardcore/SageSlayerServer/pkg/network"
	"github.com/painhardcore/SageSlayerServer/pkg/pow"
	"github.com/painhardcore/SageSlayerServer/pkg/protocol"
	"github.com/painhardcore/SageSlayerServer/pkg/ratelimiter"
	"google.golang.org/protobuf/proto"
)

// Server represents the TCP server.
type Server struct {
	Addr           string
	RateLimiter    *ratelimiter.RateLimiter
	DifficultyFunc func(requestCount float64) int
}

// NewServer creates a new Server instance.
func NewServer(addr string, halfLifeSeconds float64, difficultyFunc func(requestCount float64) int) *Server {
	rateLimiter := ratelimiter.NewRateLimiter(halfLifeSeconds)
	return &Server{
		Addr:           addr,
		RateLimiter:    rateLimiter,
		DifficultyFunc: difficultyFunc,
	}
}

// Start runs the server and listens for incoming connections.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("error starting TCP server: %v", err)
	}
	defer ln.Close()
	log.Printf("Server listening on %s", s.Addr)

	// Start cleanup goroutine
	go s.cleanupClients()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// handleConnection manages an individual client connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read/write deadlines
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Get the client's IP address
	clientIP, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		log.Printf("Error getting client IP: %v", err)
		return
	}

	// Check client action
	action := s.RateLimiter.GetClientAction(clientIP)
	switch action {
	case ratelimiter.ActionBan:
		log.Printf("Banned client %s attempted to connect", clientIP)
		time.Sleep(5 * time.Second)
		sendErrorMessage(conn, "You are temporarily banned due to suspicious activity.")
		return
	case ratelimiter.ActionIncreaseDifficulty:
		log.Printf("Client %s has increased difficulty due to errors", clientIP)
	case ratelimiter.ActionAllow:
		// Proceed normally
	}

	// Update request count and get current difficulty
	requestCount := s.RateLimiter.UpdateRequestRate(clientIP)
	difficulty := s.calculateDifficulty(requestCount, action)

	log.Printf("Client %s has a request count of %.0f. Current difficulty: %d", clientIP, requestCount, difficulty)

	// Step 1: Generate and send challenge
	challenge, err := pow.GenerateChallenge(difficulty)
	if err != nil {
		log.Printf("Error generating challenge: %v", err)
		return
	}

	if err := s.sendChallenge(conn, challenge); err != nil {
		log.Printf("Error sending challenge to %s: %v", clientIP, err)
		return
	}

	// Step 2: Receive and verify solution
	if err := s.receiveAndVerifySolution(conn, challenge, clientIP); err != nil {
		log.Printf("Error handling solution from %s: %v", clientIP, err)
		s.RateLimiter.UpdateErrorCount(clientIP)
		sendErrorMessage(conn, "Invalid solution")
		return
	}

	// Step 3: Send quote
	if err := s.sendQuote(conn); err != nil {
		log.Printf("Error sending quote to %s: %v", clientIP, err)
		return
	}
}

// sendChallenge sends the PoW challenge to the client.
func (s *Server) sendChallenge(conn net.Conn, challenge *pow.Challenge) error {
	challengeProto := &network.Challenge{
		Qx:         challenge.Qx.Bytes(),
		Qy:         challenge.Qy.Bytes(),
		Curve:      "P-256",
		Difficulty: int32(challenge.Difficulty),
	}

	challengePayload, err := proto.Marshal(challengeProto)
	if err != nil {
		return fmt.Errorf("error marshalling challenge: %v", err)
	}

	message := &network.Message{
		Type:    network.MessageType_CHALLENGE,
		Payload: challengePayload,
	}

	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshalling message: %v", err)
	}

	if err := protocol.WriteMessage(conn, data); err != nil {
		return fmt.Errorf("error sending challenge: %v", err)
	}

	return nil
}

// receiveAndVerifySolution handles receiving the solution from the client and verifying it.
func (s *Server) receiveAndVerifySolution(conn net.Conn, challenge *pow.Challenge, clientIP string) error {
	data, err := protocol.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading solution: %v", err)
	}

	message := &network.Message{}
	if err := proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("error unmarshalling message: %v", err)
	}

	if message.Type != network.MessageType_SOLUTION {
		log.Printf("Invalid message type received from %s", clientIP)
		return fmt.Errorf("invalid message type")
	}

	// Unmarshal solution
	solutionProto := &network.Solution{}
	if err := proto.Unmarshal(message.Payload, solutionProto); err != nil {
		log.Printf("Error unmarshalling solution from %s: %v", clientIP, err)
		return fmt.Errorf("invalid solution format")
	}

	// Verify solution
	err = pow.VerifySolution(challenge, solutionProto.Nonce)
	if err != nil {
		log.Printf("Invalid solution from %s: %v", clientIP, err)
		return fmt.Errorf("invalid solution")
	}

	return nil
}

// sendQuote sends a random quote to the client.
func (s *Server) sendQuote(conn net.Conn) error {
	quoteText := quotes.GetRandomQuote()
	quoteProto := &network.Quote{
		Text: quoteText,
	}
	quotePayload, err := proto.Marshal(quoteProto)
	if err != nil {
		return fmt.Errorf("error marshalling quote: %v", err)
	}

	message := &network.Message{
		Type:    network.MessageType_QUOTE,
		Payload: quotePayload,
	}
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshalling message: %v", err)
	}

	err = protocol.WriteMessage(conn, data)
	if err != nil {
		return fmt.Errorf("error sending quote: %v", err)
	}

	return nil
}

// sendErrorMessage sends an error message to the client.
func sendErrorMessage(conn net.Conn, errMsg string) {
	errorProto := &network.Error{
		Message: errMsg,
	}
	errorPayload, _ := proto.Marshal(errorProto)

	message := &network.Message{
		Type:    network.MessageType_ERROR,
		Payload: errorPayload,
	}
	data, _ := proto.Marshal(message)
	_ = protocol.WriteMessage(conn, data)
}

// cleanupClients periodically removes inactive clients from the rate limiter.
func (s *Server) cleanupClients() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		<-ticker.C
		s.RateLimiter.Cleanup(10 * time.Minute)
	}
}

func (s *Server) calculateDifficulty(requestCount float64, action ratelimiter.ClientAction) int {
	baseDifficulty := s.DifficultyFunc(requestCount)

	if action == ratelimiter.ActionIncreaseDifficulty {
		// Increase difficulty by one level
		baseDifficulty += 1
	}

	return baseDifficulty
}
