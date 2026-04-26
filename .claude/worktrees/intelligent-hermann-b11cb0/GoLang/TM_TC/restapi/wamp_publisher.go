package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	shared "scg/shared"
	"time"

	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
	"github.com/gorilla/mux"
)

type WampMessage struct {
	Topic string            `json:"topic"`
	Msg   map[string]string `json:"msg"`
}

type WampMessage1 struct {
	Topic string `json:"topic"`
	Msg   string `json:"msg"`
}

type DialogTrigger struct {
	AppName       string                 `json:"app_name"`
	DialogTitle   string                 `json:"dialogTitle"`
	DialogMessage string                 `json:"dialogMessage"`
	DialogOptions []string               `json:"dialogOptions"`
	DialogInput   map[string]interface{} `json:"dialogInput,omitempty"` // type/placeholder/default
}

var wampClient *client.Client = shared.GetWampConnection()

func publishMessage(w http.ResponseWriter, r *http.Request) {
	var wampMsg WampMessage
	var wampMsg1 WampMessage1
	msgType := 1
	err := json.NewDecoder(r.Body).Decode(&wampMsg)
	if err != nil {
		msgType = 2
		err = json.NewDecoder(r.Body).Decode(&wampMsg1)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if msgType == 1 {
		err = wampClient.Publish(wampMsg.Topic, nil, wamp.List{wampMsg.Msg}, nil)
	} else {
		err = wampClient.Publish(wampMsg1.Topic, nil, wamp.List{wampMsg1.Msg}, nil)
	}

	if err != nil {
		log.Println("Error publishing message:", err)
		http.Error(w, "Failed to publish message", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Message published successfully")
}

func triggerDialog(w http.ResponseWriter, r *http.Request) {
	// Check if the WAMP client is connected
	wampClient := shared.GetWampConnection()
	if wampClient == nil || !wampClient.Connected() {
		log.Println("WAMP client is not connected!")
		http.Error(w, "WAMP backend unavailable", http.StatusServiceUnavailable)
		return
	}

	// Read and parse the JSON request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading request body:", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var trigger DialogTrigger
	err = json.Unmarshal(body, &trigger)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if trigger.AppName == "" || trigger.DialogTitle == "" || trigger.DialogMessage == "" {
		http.Error(w, "Missing required fields: app_name, dialogTitle, dialogMessage", http.StatusBadRequest)
		return
	}

	// Build the WAMP kwargs for the frontend
	kwargs := wamp.Dict{
		"app_name":      trigger.AppName,
		"dialogTitle":   trigger.DialogTitle,
		"dialogMessage": trigger.DialogMessage,
		"dialogOptions": trigger.DialogOptions,
	}
	if trigger.DialogInput != nil {
		kwargs["dialogInput"] = trigger.DialogInput
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call the WAMP RPC and wait for the user's dialog response
	result, err := wampClient.Call(ctx, "com.app.show_form_dialog", nil, nil, kwargs, nil)
	if err != nil {
		log.Println("Error calling dialog RPC:", err)
		http.Error(w, "Failed to call dialog: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the user's dialog response to the REST client (as JSON)
	if len(result.Arguments) > 0 {
		// Validate that the response is JSON serializable
		if _, ok := result.Arguments[0].(map[string]interface{}); !ok {
			if _, ok := result.Arguments[0].([]interface{}); !ok {
				log.Println("Unexpected response format from dialog RPC")
				http.Error(w, "Invalid response format from dialog", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		encErr := json.NewEncoder(w).Encode(result.Arguments[0])
		if encErr != nil {
			log.Println("Error encoding response JSON:", encErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func registerRoutesForWampPublisher(r *mux.Router) {
	r.HandleFunc("/publish_to_wamp", publishMessage).Methods("POST")
	r.HandleFunc("/trigger_ui_dialog", triggerDialog).Methods("POST")
}
