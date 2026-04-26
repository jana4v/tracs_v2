package Telemetry

import (
	"context"
	"fmt"
	shared "scg/shared"
	"strings"
)

func TmApp(umacsDataServerIP string) {
	r := shared.GetRedisConnection()
	ctx := context.Background()

	DATA_SERVER_IP := strings.TrimSpace(umacsDataServerIP)
	if DATA_SERVER_IP == "" {
		DATA_SERVER_IP = "127.0.0.1"
		SCC_DATA_SERVER_IP, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_SERVER_IP").Result()
		if err != nil {
			r.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_SERVER_IP", "172.20.5.xx").Result()
		} else if strings.TrimSpace(SCC_DATA_SERVER_IP) != "" {
			DATA_SERVER_IP = SCC_DATA_SERVER_IP
		}
	}

	data_streams, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_STREAMS").Result()

	if err != nil {
		//TM1,TM2,SMON1,ADC1,PTM1,PTM2,SMON2,ADC2
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_STREAMS", "TM1,TM2,SMON1").Result()
		data_streams = "TM1,TM2,SMON1"
	}
	r.Close()
	streams := strings.Split(data_streams, ",")
	fmt.Println(streams)
	if contains(streams, "TM1") {
		go TmSubscriber("TM1", DATA_SERVER_IP+":"+shared.TM1_PORT)
	}
	if contains(streams, "TM2") {
		go TmSubscriber("TM2", DATA_SERVER_IP+":"+shared.TM2_PORT)
	}

	if contains(streams, "SMON1") {
		go SmonSubscriber("SMON1", DATA_SERVER_IP+":"+shared.SMON1_PORT)
	}

	if contains(streams, "ADC1") {
		go ADCSubscriber("ADC1", DATA_SERVER_IP+":"+shared.ADC1_PORT)
	}

	if contains(streams, "SMON2") {
		go SmonSubscriber("SMON2", DATA_SERVER_IP+":"+shared.SMON2_PORT)
	}

	if contains(streams, "ADC2") {
		go ADCSubscriber("ADC2", DATA_SERVER_IP+":"+shared.ADC2_PORT)
	}

	if contains(streams, "PTM1") {
		go PtmSubscriber("PTM1", DATA_SERVER_IP+":"+shared.PTM1_PORT)
	}

	if contains(streams, "PTM2") {
		go PtmSubscriber("PTM2", DATA_SERVER_IP+":"+shared.PTM1_PORT)
	}

	if contains(streams, "TC") {
		go TcSent(DATA_SERVER_IP + ":" + shared.TC_SENT_PORT)
	}

	if contains(streams, "ACSS_FILE_STS") {
		go acss_tcFileStatus()
	}
	//go TcSent(DATA_SERVER_IP + ":" + shared.TC_SENT_PORT)
	// go pollWebSocketForStatuses(ctx, r)
	go UserDefinedTm()
	go InjectTm()
	go RedisToWampMessagePublisher()

}
