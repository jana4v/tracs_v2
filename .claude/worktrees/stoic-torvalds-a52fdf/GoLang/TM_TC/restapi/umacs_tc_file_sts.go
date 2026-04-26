package restapi

import (
	"context"
	"log"

	shared "scg/shared"
	"strconv"
	"time"

	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
	"github.com/go-redis/redis/v8"
)

// Function to poll WebSocket server for file statuses
func pollForTcFileStatuses(ctx context.Context, rdb *redis.Client) {
	var wampClient *client.Client = shared.GetWampConnection()
	var wamp_topic string = "com.tc_file.status"
	var tc_file_status map[string]string = map[string]string{
		"summary":  "File status",
		"status":   "",
		"progress": "",
	}

	var file_status map[string]string = map[string]string{}

	ticker := time.NewTicker(2000 * time.Millisecond)
	defer ticker.Stop()
	//var ws *websocket.Conn
	// defer ws.Close()
	for range ticker.C {
		files, err := rdb.HGetAll(ctx, shared.RedisKeys.TC_FILES_STATUS).Result()
		if err != nil {
			//log.Printf("Failed to fetch file statuses from Redis: %v", err)
			continue
		}
		for fileName, status := range files {
			if contains(status, dont_check_status) {
				continue
			}
			if _, exist := file_status[fileName]; !exist {
				file_status[fileName] = ""
			}
			tc_file_status["summary"] = "File:" + fileName + " Status.."
			// Fetch the IP address from Redis using HGet
			var tcReq Request
			tcReq.ProcName = fileName
			res, err := tcReq.Get_file_status()
			if err != nil {
				log.Printf("Failed to fetch file status from TC: %v for the File:%v", err, fileName)
				continue
			}

			if res.ExeStatus != not_available {
				if file_status[fileName] != res.ExeStatus {
					channel := "TC_FILE_EXECUTION_STATUS"
					rdb.Publish(ctx, channel, fileName+":"+res.ExeStatus).Err()
					rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, fileName, res.ExeStatus).Result()
					rdb.HDel(ctx, "TC_FILES_NOT_AVAILABLE_COUNT", fileName).Result()
					if res.ExeStatus != "in-progress" && res.ExeStatus != "success" {
						tc_file_status["status"] = res.ExeStatus
						tc_file_status["progress"] = "70"
						if res.ExeStatus == "failure" {
							tc_file_status["progress"] = "0"
						}
						wampClient.Publish(wamp_topic, nil, wamp.List{tc_file_status}, nil)
					}
				}
			} else {
				// Increment the not-available count
				countStr, err := rdb.HGet(ctx, "TC_FILES_NOT_AVAILABLE_COUNT", fileName).Result()
				if err == redis.Nil {
					countStr = "0"
				} else if err != nil {
					log.Printf("Failed to fetch not-available count from Redis: %v", err)
					continue
				}

				count, _ := strconv.Atoi(countStr)
				count++

				// If count exceeds the max iterations, remove the file
				if count >= 10 {
					tc_file_status["status"] = "Execution Status Not Available"
					tc_file_status["progress"] = "0"
					wampClient.Publish(wamp_topic, nil, wamp.List{tc_file_status}, nil)
					log.Printf("Removing file %v due to not-available status for %d iterations", fileName, 10)
					rdb.HDel(ctx, shared.RedisKeys.TC_FILES_STATUS, fileName).Result()
					rdb.HDel(ctx, shared.RedisKeys.TC_FILES_START_TIME, fileName).Result()
					rdb.HDel(ctx, "TC_FILES_NOT_AVAILABLE_COUNT", fileName).Result()
					rdb.Publish(ctx, "TC_FILE_REMOVED", fileName).Err()
					//rdb.Del(ctx, fileName).Err()
				} else {
					rdb.HSet(ctx, "TC_FILES_NOT_AVAILABLE_COUNT", fileName, count).Result()
				}
			}

			if res.ExeStatus == "success" {
				tc_file_status["status"] = "Execution Completed"
				tc_file_status["progress"] = "100"
				wampClient.Publish(wamp_topic, nil, wamp.List{tc_file_status}, nil)
				// Calculate execution time
				startTimeStr, _ := rdb.HGet(ctx, shared.RedisKeys.TC_FILES_START_TIME, fileName).Result()
				startTime, err := time.Parse(time.RFC3339, startTimeStr)
				if err == nil {
					executionTime := time.Since(startTime).Seconds()
					// Fetch existing execution time and add the new execution time
					existingTimeStr, err := rdb.Get(ctx, shared.RedisKeys.TC_FILES_EXECUTION_TIME).Result()
					if err != nil {
						existingTimeStr = "0"
					}
					var existingTime float64
					if existingTimeStr != "" {
						existingTime, err = strconv.ParseFloat(existingTimeStr, 64)
						if err != nil {
							existingTime = 0
						}
					}
					totalExecutionTime := existingTime + executionTime
					// Update execution time in Redis
					rdb.Set(ctx, shared.RedisKeys.TC_FILES_EXECUTION_TIME, totalExecutionTime, 0)
				}
				// Remove the file from Redis
				rdb.HDel(ctx, shared.RedisKeys.TC_FILES_START_TIME, fileName)
				channel := "TC_FILE_EXECUTION_COMPLETED"
				rdb.Publish(ctx, channel, fileName).Err()
			} else if res.ExeStatus == "in-progress" {
				channel := "TC_FILE_EXECUTION_STARTED"
				rdb.Publish(ctx, channel, fileName).Err()
				if file_status[fileName] != res.ExeStatus {
					tc_file_status["status"] = "Execution is in progress"
					tc_file_status["progress"] = "80"
					wampClient.Publish(wamp_topic, nil, wamp.List{tc_file_status}, nil)
				}
				previous_status := rdb.HGet(ctx, shared.RedisKeys.TC_FILES_STATUS, fileName).Val()
				if previous_status == "in-progress" {
					startTimeStr, _ := rdb.HGet(ctx, shared.RedisKeys.TC_FILES_START_TIME, fileName).Result()
					startTime, err := time.Parse(time.RFC3339, startTimeStr)
					if err == nil {
						executionTime := time.Since(startTime).Seconds()
						// Fetch existing execution time and add the new execution time
						existingTimeStr, err := rdb.Get(ctx, shared.RedisKeys.TC_FILES_WAIT_TIME).Result()
						if err != nil {
							existingTimeStr = "0"
						}

						var existingTime float64
						if existingTimeStr != "" {
							existingTime, err = strconv.ParseFloat(existingTimeStr, 64)
							if err != nil {
								existingTime = 0
							}
						}
						totalExecutionTime := existingTime + executionTime
						// Update execution time in Redis
						rdb.Set(ctx, shared.RedisKeys.TC_FILES_WAIT_TIME, totalExecutionTime, 0).Result()
					}
				}
				rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, fileName, res.ExeStatus).Result()
			}

			file_status[fileName] = res.ExeStatus

		}
	}
}
