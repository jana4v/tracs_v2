package main

import (
	"flag"
	"strings"
	"sync"

	api "scg/restapi"
	shared "scg/shared"

	tm "scg/tm"

	wampApi "scg/wampApi"
)

var wg sync.WaitGroup

func main() {
	shared.ReadTomlConfigFile()

	umacsDataServerIPFlag := flag.String("umacs_data_server_ip", "", "UMACS data server IP")
	flag.Parse()
	umacsDataServerIP := strings.TrimSpace(*umacsDataServerIPFlag)
	if umacsDataServerIP == "" {
		umacsDataServerIP = strings.TrimSpace(shared.GetTomlConfigValue("umacs_data_server_ip"))
	}
	wg.Add(1)
	go api.RestApiServer()
	go tm.TmApp(umacsDataServerIP)
	go wampApi.InitWampAPI()
	wg.Wait()
}
