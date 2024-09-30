package server

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/painhardcore/SageSlayerServer/internal/quotes"
	"github.com/painhardcore/SageSlayerServer/pkg/network"
	"github.com/painhardcore/SageSlayerServer/pkg/pow"
	"github.com/painhardcore/SageSlayerServer/pkg/protocol"
	"github.com/painhardcore/SageSlayerServer/pkg/ratelimiter"
	"github.com/panjf2000/gnet/v2"
	"google.golang.org/protobuf/proto"
)

// Server represents the gnet server.
type Server struct {
	Addr           string
	RateLimiter    *ratelimiter.RateLimiter
	DifficultyFunc func(requestCount float64) int
	Challenges     map[int]*pow.Challenge // Keyed by connection ID (fd)
	Mu             sync.RWMutex           // Mutex for protecting the Challenges map
}

// NewServer creates a new Server instance.
func NewServer(addr string, halfLifeSeconds float64, difficultyFunc func(requestCount float64) int) *Server {
	rateLimiter := ratelimiter.NewRateLimiter(halfLifeSeconds)
	return &Server{
		Addr:           addr,
		RateLimiter:    rateLimiter,
		DifficultyFunc: difficultyFunc,
		Challenges:     make(map[int]*pow.Challenge), // Initialize the Challenges map
		Mu:             sync.RWMutex{},               // Mutex is automatically ready after declaration
	}
}

// EventServer implements gnet.EventHandler
type EventServer struct {
	*gnet.BuiltinEventEngine
	server *Server
}

// OnBoot is called when the server starts.
func (es *EventServer) OnBoot(eng gnet.Engine) gnet.Action {
	log.Printf("Server listening on %s", es.server.Addr)
	go es.server.cleanupClients()
	return gnet.None
}

// OnShutdown is called when the server shuts down.
func (es *EventServer) OnShutdown(eng gnet.Engine) {
	log.Println("Server shutting down")
}

func (es *EventServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	clientIP, _, err := net.SplitHostPort(c.RemoteAddr().String())
	if err != nil {
		log.Printf("Error getting client IP: %v", err)
		return
	}
	// Check client action with RateLimiter (using IP to track by IP)
	actionType := es.server.RateLimiter.GetClientAction(clientIP)
	switch actionType {
	case ratelimiter.ActionBan:
		log.Printf("Banned client %s attempted to connect", clientIP)
		time.Sleep(5 * time.Second)
		return sendErrorMessage("You are temporarily banned due to suspicious activity."), gnet.Close
	case ratelimiter.ActionIncreaseDifficulty:
		log.Printf("Client %s has increased difficulty due to errors", clientIP)
	case ratelimiter.ActionAllow:
		// Proceed normally
	}

	// Update request count and get difficulty
	requestCount := es.server.RateLimiter.UpdateRequestRate(clientIP)
	difficulty := es.server.calculateDifficulty(requestCount, actionType)
	log.Printf("Client %s has a request count of %.0f. Current difficulty: %d", clientIP, requestCount, difficulty)

	// Generate challenge
	challenge, err := pow.GenerateChallenge(difficulty)
	if err != nil {
		log.Printf("Error generating challenge: %v", err)
		return nil, gnet.Close // Close if there's an error
	}

	// Store the challenge for the connection (using the connection ID or another identifier)
	es.server.Mu.Lock()
	es.server.Challenges[c.Fd()] = challenge
	es.server.Mu.Unlock()

	// Send the challenge to the client
	err = es.server.sendChallenge(c, challenge)
	if err != nil {
		log.Printf("Error sending challenge: %v", err)
		return nil, gnet.Close // Close if there's an error sending the challenge
	}

	return nil, gnet.None // Keep the connection open
}

func (es *EventServer) OnTraffic(c gnet.Conn) gnet.Action {
	connID := c.Fd() // Unique connection ID
	clientIP := c.RemoteAddr().String()

	// Retrieve the stored challenge for the connection
	es.server.Mu.RLock()
	challenge, exists := es.server.Challenges[connID]
	es.server.Mu.RUnlock()
	if !exists {
		log.Printf("No challenge found for connection %d", connID)
		return gnet.Close
	}

	// Read the message using the protocol's length-prefixed method
	data, err := protocol.ReadMessageGnet(c)
	if err != nil {
		log.Printf("Error reading message from connection %d: %v", connID, err)
		return gnet.Close // Close only if there's an error reading the message
	}
	if data == nil {
		return gnet.None // message not fully received yet, keep the connection open
	}

	// Verify the solution
	err = es.server.receiveAndVerifySolution(data, challenge, clientIP)
	if err != nil {
		log.Printf("Error handling solution from connection %d: %v", connID, err)
		es.server.RateLimiter.UpdateErrorCount(clientIP)
		c.AsyncWrite(sendErrorMessage("Invalid solution"), nil)
		return gnet.Close // Close if solution is invalid
	}

	// Send quote to the client
	quoteData, err := es.server.sendQuote()
	if err != nil {
		log.Printf("Error sending quote to connection %d: %v", connID, err)
		return gnet.Close // Close if there's an error sending the quote
	}

	// Send quote data asynchronously
	err = protocol.WriteMessageGnet(c, quoteData)
	if err != nil {
		log.Printf("Error sending quote to connection %d: %v", connID, err)
		return gnet.Close // Close if there's an error sending the quote
	}
	return gnet.None // Keep the connection open
}

// OnClose is called when a connection is closed.
func (es *EventServer) OnClose(c gnet.Conn, err error) gnet.Action {
	connID := c.Fd() // Unique connection ID
	// Remove the challenge associated with this connection
	es.server.Mu.Lock()
	delete(es.server.Challenges, connID)
	es.server.Mu.Unlock()
	return gnet.None
}

// Start runs the gnet server.
func (s *Server) Start() error {
	eventServer := &EventServer{server: s}
	err := gnet.Run(eventServer, fmt.Sprintf("tcp://%s", s.Addr), gnet.WithMulticore(true))
	if err != nil {
		return fmt.Errorf("error starting gnet server: %v", err)
	}
	return nil
}

// sendChallenge prepares and sends the PoW challenge to the client with size-prefixed data.
func (s *Server) sendChallenge(c gnet.Conn, challenge *pow.Challenge) error {
	// Marshal the challenge to protobuf
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

	messageData, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshalling message: %v", err)
	}

	// Write the length-prefixed message
	return protocol.WriteMessage(c, messageData)
}

// receiveAndVerifySolution handles receiving the solution from the client and verifying it.
func (s *Server) receiveAndVerifySolution(data []byte, challenge *pow.Challenge, clientIP string) error {
	if challenge == nil {
		return fmt.Errorf("challenge is nil")
	}

	message := &network.Message{}
	if err := proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("error unmarshalling message: %v", err)
	}

	if message.Type != network.MessageType_SOLUTION {
		return fmt.Errorf("invalid message type")
	}

	// Unmarshal solution
	solutionProto := &network.Solution{}
	if err := proto.Unmarshal(message.Payload, solutionProto); err != nil {
		return fmt.Errorf("invalid solution format")
	}

	// Verify solution, ensure challenge is not nil
	err := pow.VerifySolution(challenge, solutionProto.Nonce)
	if err != nil {
		return fmt.Errorf("invalid solution: %v", err)
	}

	return nil
}

// sendQuote sends a random quote to the client.
func (s *Server) sendQuote() ([]byte, error) {
	quoteText := quotes.GetRandomQuote()
	quoteProto := &network.Quote{
		Text: quoteText,
	}
	quotePayload, err := proto.Marshal(quoteProto)
	if err != nil {
		return nil, fmt.Errorf("error marshalling quote: %v", err)
	}

	message := &network.Message{
		Type:    network.MessageType_QUOTE,
		Payload: quotePayload,
	}

	return proto.Marshal(message)
}

// sendErrorMessage sends an error message to the client.
func sendErrorMessage(errMsg string) []byte {
	errorProto := &network.Error{
		Message: errMsg,
	}
	errorPayload, _ := proto.Marshal(errorProto)

	message := &network.Message{
		Type:    network.MessageType_ERROR,
		Payload: errorPayload,
	}
	data, _ := proto.Marshal(message)
	return data
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
		baseDifficulty += 1
	}

	return baseDifficulty
}
