package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// CommandRequestPoll represents the structure of the incoming request
type CommandRequestPoll struct {
	UUID        string `json:"uuid"`
	Value       string `json:"value"`
	ChannelName string `json:"channel_name"`
}

// RequestState tracks the state of each request
type RequestState struct {
	mu              sync.Mutex
	receivedMsgs    map[string]bool
	activeListeners map[string]context.CancelFunc
	lastAccessTime  map[string]time.Time
}

// NewRequestState initializes a new RequestState
func NewRequestState() *RequestState {
	return &RequestState{
		receivedMsgs:    make(map[string]bool),
		activeListeners: make(map[string]context.CancelFunc),
		lastAccessTime:  make(map[string]time.Time),
	}
}

// SetMessageReceived marks a message as received
func (rs *RequestState) SetMessageReceived(requestID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.receivedMsgs[requestID] = true
}

// IsMessageReceived checks if a message has been received
func (rs *RequestState) IsMessageReceived(requestID string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.receivedMsgs[requestID]
}

// ResetMessageReceived resets the received message state
func (rs *RequestState) ResetMessageReceived(requestID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.receivedMsgs, requestID)
}

// SetListenerActive marks a listener as active
func (rs *RequestState) SetListenerActive(requestID string, cancelFunc context.CancelFunc) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.activeListeners[requestID] = cancelFunc
}

// ResetListenerActive resets the listener state
func (rs *RequestState) ResetListenerActive(requestID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if cancelFunc, ok := rs.activeListeners[requestID]; ok {
		cancelFunc()
		delete(rs.activeListeners, requestID)
	}
}

// IsListenerActive checks if a listener is active
func (rs *RequestState) IsListenerActive(requestID string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	_, exists := rs.activeListeners[requestID]
	return exists
}

// UpdateLastAccessTime updates the last access time for a request
func (rs *RequestState) UpdateLastAccessTime(requestID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.lastAccessTime[requestID] = time.Now()
}

// IsTimeoutExpired checks if the timeout has expired for a request
func (rs *RequestState) IsTimeoutExpired(requestID string, timeout time.Duration) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	lastAccess, exists := rs.lastAccessTime[requestID]
	if !exists {
		return false
	}
	return time.Since(lastAccess) > timeout
}

var requestState = NewRequestState()

// handleCommandPoll handles both setting up the listener and polling for the command
func handleCommandPoll(w http.ResponseWriter, r *http.Request) {
	var req CommandRequestPoll
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Generate a new UUID if not provided
	if req.UUID == "" {
		req.UUID = "no_uuid_provided"
	}

	requestID := req.UUID
	command := req.Value
	channelName := req.ChannelName

	if command == "" || channelName == "" {
		http.Error(w, "Command, ChannelName, and UUID fields are required", http.StatusBadRequest)
		return
	}

	if requestState.IsTimeoutExpired(requestID, 20*time.Second) {
		requestState.ResetMessageReceived(requestID) // Reset the message state
		requestState.ResetListenerActive(requestID)  // Reset the listener state
		fmt.Printf("Resetting request state for UUID: %s\n", requestID)
	}

	// Update the last access time for the request
	requestState.UpdateLastAccessTime(requestID)

	if !requestState.IsListenerActive(requestID) {
		// If no listener is active, start a new one in a goroutine with a cancelable context
		ctx, cancel := context.WithCancel(context.Background())
		requestState.SetListenerActive(requestID, cancel)
		go listenToChannelPoll(ctx, requestID, channelName, command)
	}

	// Poll the state to see if the message has been received
	if requestState.IsMessageReceived(requestID) {
		fmt.Fprintf(w, `{"status": true, "uuid": "%s"}`, requestID)
		requestState.ResetMessageReceived(requestID) // Reset the message state
		requestState.ResetListenerActive(requestID)  // Reset the listener state
	} else {
		fmt.Fprintf(w, `{"status": false, "uuid": "%s"}`, requestID)
	}
}

// listenToChannelPoll listens for the specified value on the Redis channel
func listenToChannelPoll(ctx context.Context, requestID, channelName, command string) {
	pubsub := rdb.Subscribe(ctx, channelName)
	defer func() {
		pubsub.Unsubscribe(ctx)
		pubsub.Close()
	}()

	ch := pubsub.Channel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			if msg.Payload == command {
				fmt.Printf("Received expected message for request ID: %s\n", requestID)
				requestState.SetMessageReceived(requestID) // Update the message state
				return
			}
		case <-ticker.C:
			if requestState.IsTimeoutExpired(requestID, 20*time.Second) {
				requestState.ResetMessageReceived(requestID) // Reset the message state
				requestState.ResetListenerActive(requestID)  // Reset the listener state
				fmt.Printf("Listener timed out for request ID: %s\n", requestID)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

/*This code creates a REST API endpoint that allows clients to send a request with a UUID,
expected value, and Redis channel name. The server then listens in the background for that
specific value to be published on the Redis channel. Clients can poll the same endpoint
repeatedly to check if the expected message has arrived. Once the message is received,
the server responds with a success status. The system also handles timeouts (20 seconds)
to clean up stale requests and uses thread-safe operations to manage multiple concurrent clients efficiently.
*/

func registerRoutesForRedisChannelValuePoling(r *mux.Router) {
	r.HandleFunc("/poll_for_redis_channel_value", handleCommandPoll).Methods("POST")
}
