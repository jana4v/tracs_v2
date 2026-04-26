package Telemetry

import (
	"context"
	"fmt"
	"net"
	"regexp"
	shared "scg/shared"
	"strconv"
	"time"

	"github.com/gammazero/nexus/v3/wamp"
)

func acss_tcFileStatus() {
	wamp_tc_file_status := "com.tc_file.status"
	var status_msg = make(map[string]string)
	ctx := context.Background()
	publisher := shared.GetWampConnection()
	defer publisher.Close()
	r := shared.GetRedisConnection()
	defer r.Close()
	channel := "TC_FILE_EXECUTION_COMPLETED"
	port, _ := r.HGet(context.Background(), "ENV_VARIABLES", "TC_FILE_STATUS_PORT").Result()
	portnum, _ := strconv.Atoi(port)
	ServerConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0, 0, 0, 0}, Port: portnum, Zone: ""})
	defer ServerConn.Close()
	//For sts forward
	ip, _ := r.HGet(context.Background(), "ENV_VARIABLES", "FILE_STS_FORWARD_IP").Result()
	if ip == "" {
		ip = "172.20.5.222"
	}
	// udpAddr := &net.UDPAddr{
	// 	IP:   net.ParseIP(ip),
	// 	Port: 5555,
	// }
	// fc, _ := net.DialUDP("udp", nil, udpAddr)
	// defer fc.Close()
	// end
	re_try, _ := regexp.Compile(`TRY`)
	re_wait, _ := regexp.Compile(`WAIT`)
	re_rejected, _ := regexp.Compile(`REJECTED`)
	re_accepted, _ := regexp.Compile(`ACCEPTED`)
	re_progress, _ := regexp.Compile(`PROGRESS`)
	re_executionOver, _ := regexp.Compile(`EXECUTION OVER`)
	re_aborted, _ := regexp.Compile(`ABORTED`)
	status_msg["topic"] = "com.tc_file.status"
	buf := make([]byte, 1024)
	for {
		n, _, _ := ServerConn.ReadFromUDP(buf)
		// _, ferr := fc.Write(buf[:n])
		// if ferr != nil {
		// 	logger.Println("Failed to forward message", ferr)
		// }
		status := string(buf[2:n])
		fmt.Println(status)
		try := re_try.MatchString(status)
		wait := re_wait.MatchString(status)
		rejected := re_rejected.MatchString(status)
		accepted := re_accepted.MatchString(status)
		progress := re_progress.MatchString(status)
		executionOver := re_executionOver.MatchString(status)
		aborted := re_aborted.MatchString(status)

		fileName, err := r.HGet(ctx, "PAYLOAD_TC_FILE_STS", "TC_FILE_NAME").Result()
		//fmt.Println(fileName, err)
		//file_type, _ := r.HGet(ctx, "PAYLOAD_TC_FILE_STS", "TC_FILE_TYPE").Result()
		//{"topic":"com.gisat.status","summary":"File Generation In Progress for Config Numbers: 1","status":"Active Configurations:[]","progress":"50"}}
		if err == nil {
			if try || rejected || wait {
				_ = r.Set(ctx, "xx"+fileName, "REJECTED", 30*time.Minute).Err()
				_ = r.Set(ctx, "STS", "REJECTED", 30*time.Minute).Err()
				status_msg["summary"] = "File Execution Rejected in SCC"
				status_msg["status"] = ""
				status_msg["progress"] = "0"
				//fmt.Println(status_msg)
				_ = publisher.Publish(wamp_tc_file_status, nil, wamp.List{status_msg}, nil)
			}
			if aborted {
				status_msg["summary"] = "File Execution Aborted in SCC"
				status_msg["status"] = ""
				status_msg["progress"] = "75"
				_ = r.Set(ctx, "xx"+fileName, "ABORTED", 30*time.Minute).Err()
				_ = r.Set(ctx, "STS", "ABORTED", 30*time.Minute).Err()
				_ = publisher.Publish(wamp_tc_file_status, nil, wamp.List{status_msg}, nil)
			}

			if accepted {
				status_msg["summary"] = "File Execution Accepted in SCC"
				status_msg["status"] = ""
				status_msg["progress"] = "70"
				_ = r.Set(ctx, "xx"+fileName, "ACCEPTED", 30*time.Minute).Err()
				_ = r.Set(ctx, "STS", "ACCEPTED", 30*time.Minute).Err()
				_ = publisher.Publish(wamp_tc_file_status, nil, wamp.List{status_msg}, nil)
			}
			if progress {
				status_msg["summary"] = "File Execution in Progress in SCC"
				status_msg["status"] = ""
				status_msg["progress"] = "80"
				_ = r.Set(ctx, "xx"+fileName, "IN_PROGRESS", 600*time.Minute).Err()
				_ = r.Set(ctx, "STS", "IN_PROGRESS", 600*time.Minute).Err()
				_ = publisher.Publish(wamp_tc_file_status, nil, wamp.List{status_msg}, nil)

			}
			if executionOver {
				status_msg["summary"] = "File Execution Completed"
				status_msg["status"] = ""
				status_msg["progress"] = "100"
				_ = r.Set(ctx, "xx"+fileName, "COMPLETED", 30*time.Minute).Err()
				_ = r.Set(ctx, "STS", "COMPLETED", 30*time.Minute).Err()
				_ = publisher.Publish(wamp_tc_file_status, nil, wamp.List{status_msg}, nil)
				err := r.Publish(ctx, channel, fileName).Err()
				if err != nil {
					logger.Println("Error publishing message:", err)
				}
			}
		}
	}
}
