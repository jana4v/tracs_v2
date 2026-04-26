package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	redis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

type Item struct {
	Priority int         `json:"Priority"`
	Value    interface{} `json:"Value"`
}

// QueueManager manages multiple priority queues using Redis.
type QueueManager struct {
	lock sync.Mutex
}

func NewQueueManager() *QueueManager {
	return &QueueManager{}
}

// getRedisClient returns the shared Redis connection
func (qm *QueueManager) getRedisClient() *redis.Client {
	return rdb // Use the shared Redis client from main.go
}

// getQueueKey returns the Redis sorted set key for a queue
func (qm *QueueManager) getQueueKey(queueName string) string {
	return fmt.Sprintf("pqueue:%s", queueName)
}

// Enqueue adds an item to the queue (stored in Redis sorted set)
func (qm *QueueManager) Enqueue(queueName string, item Item) error {
	qm.lock.Lock()
	defer qm.lock.Unlock()

	ctx := context.Background()

	// Serialize the item value to JSON
	valueBytes, err := json.Marshal(item.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal item value: %w", err)
	}

	// Store in Redis sorted set: score = priority, member = JSON value
	key := qm.getQueueKey(queueName)
	err = qm.getRedisClient().ZAdd(ctx, key, &redis.Z{
		Score:  float64(item.Priority),
		Member: string(valueBytes),
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to enqueue item: %w", err)
	}

	return nil
}

// Dequeue removes and returns the item with the lowest priority
func (qm *QueueManager) Dequeue(queueName string) (*Item, error) {
	qm.lock.Lock()
	defer qm.lock.Unlock()

	ctx := context.Background()
	key := qm.getQueueKey(queueName)

	// Get the item with the lowest score (highest priority)
	result, err := qm.getRedisClient().ZPopMin(ctx, key, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue item: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // Queue is empty
	}

	// Parse the result
	member := result[0].Member.(string)
	priority := int(result[0].Score)

	var value interface{}
	err = json.Unmarshal([]byte(member), &value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item value: %w", err)
	}

	return &Item{
		Priority: priority,
		Value:    value,
	}, nil
}

// Clear removes all items from a queue
func (qm *QueueManager) Clear(queueName string) error {
	qm.lock.Lock()
	defer qm.lock.Unlock()

	ctx := context.Background()
	key := qm.getQueueKey(queueName)
	err := qm.getRedisClient().Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	return nil
}

// GetQueueSize returns the number of items in a queue
func (qm *QueueManager) GetQueueSize(queueName string) (int64, error) {
	ctx := context.Background()
	key := qm.getQueueKey(queueName)
	return qm.getRedisClient().ZCard(ctx, key).Result()
}

var manager = NewQueueManager()

func enqueueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]

	var item Item
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = manager.Enqueue(queueName, item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func dequeueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]

	item, err := manager.Dequeue(queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if item == nil {
		http.Error(w, "Queue is empty or does not exist", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(item)
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queueName := vars["queueName"]

	err := manager.Clear(queueName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func registerRoutesForPriorityQueue(r *mux.Router) {
	r.HandleFunc("/enqueue/{queueName}", enqueueHandler).Methods("POST")
	r.HandleFunc("/dequeue/{queueName}", dequeueHandler).Methods("GET")
	r.HandleFunc("/clear/{queueName}", clearHandler).Methods("POST")
}
