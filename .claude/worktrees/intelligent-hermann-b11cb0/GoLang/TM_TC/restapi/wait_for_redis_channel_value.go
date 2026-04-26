package restapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// CommandRequest represents the structure of the incoming request
type CommandRequest struct {
	Value       string `json:"value"`
	ChannelName string `json:"channel_name"`
	Timeout     int    `json:"timeout,omitempty"`
}

func subscribeToCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	command := req.Value
	channelName := req.ChannelName
	timeout := req.Timeout
	if command == "" || channelName == "" {
		http.Error(w, "Command and ChannelName fields are required", http.StatusBadRequest)
		return
	}

	// Default timeout to 3600 seconds if not provided
	if timeout == 0 {
		timeout = 3600
	}

	responseChannel := make(chan string)
	go listenToChannel(channelName, command, responseChannel)

	select {
	case response := <-responseChannel:
		fmt.Fprintf(w, "Received command: %s\n", response)
	case <-time.After(time.Duration(timeout) * time.Second): // Use the timeout value from the request
		http.Error(w, "Timeout waiting for command", http.StatusRequestTimeout)
	}
}

func listenToChannel(channelName string, command string, responseChannel chan string) {
	pubsub := rdb.Subscribe(ctx, channelName)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		if msg.Payload == command {
			responseChannel <- msg.Payload
			break
		}
	}
}

func registerRoutesForRedisChannelValue(r *mux.Router) {
	r.HandleFunc("/wait_for_redis_channel_value", subscribeToCommand).Methods("POST")
}
