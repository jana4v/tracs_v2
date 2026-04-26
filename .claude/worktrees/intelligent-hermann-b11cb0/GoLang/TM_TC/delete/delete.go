package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-redis/redis/v8"
)

func GetRedisConnection() *redis.Client {
	r := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := r.Ping(context.Background()).Result()
	if err != nil {
		fmt.Println("Failed to Connect Redis Server")
		fmt.Println(err)
	}
	return r
}

func extractVariables2(expression string) []string {
	prefixes := []string{"tm1.", "tm2.", "ptm1.", "ptm2.", "dtm.", "smon.", "adc."}
	var variables []string

	for _, prefix := range prefixes {
		// Match variables in the format: prefix.something
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(prefix) + `\w+\b`)
		matches := re.FindAllString(expression, -1)
		variables = append(variables, matches...)

		// Match variables in the format: prefix["something"]
		re = regexp.MustCompile(`\b` + regexp.QuoteMeta(prefix) + `\[\s*\"[^\"]+\"\s*\]`)
		matches = re.FindAllString(expression, -1)
		for _, match := range matches {
			// Clean up the match to remove the brackets and quotes
			cleaned := match[:len(prefix)+1] + match[len(prefix)+2:len(match)-2]
			variables = append(variables, cleaned)
		}

		// Match variables in the format: prefix["something with spaces or special characters"]
		re = regexp.MustCompile(`\b` + regexp.QuoteMeta(prefix) + `\[\s*\"[^\"]+\"\s*\]`)
		matches = re.FindAllString(expression, -1)
		for _, match := range matches {
			// Clean up the match to remove the brackets and quotes
			cleaned := strings.Replace(match, "[\"", ".", 1)
			cleaned = strings.Replace(cleaned, "\"]", "", 1)
			variables = append(variables, cleaned)
		}
	}
	return variables
}

func stringInArray(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

func main() {
	r := GetRedisConnection()

	ctx := context.Background()
	data_streams, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_STREAMS").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_STREAMS", "TM1,TM2,SMON1,ADC1,PTM1,PTM2,SMON2,ADC2").Result()
		data_streams = "TM1,TM2,SMON1,ADC1,PTM1,PTM2,SMON2,ADC2"
	}
	streams := strings.Split(data_streams, ",")
	fmt.Println(stringInArray(streams, "XTM1"))
	for i, stream := range strings.Split(data_streams, ",") {
		println(i, stream)
	}
	// if err != nil {
	// 	sftpPort = "22" // Default SFTP port if not provided in Redis
	// 	r.HSet(ctx, "ENV_VARIABLES", "SFTP_PORT", 22).Result()
	// }

	// fmt.Println(sftpPort)
	// s1, e1 := r.Get(ctx, "xx").Result()
	// if e1 != nil {
	// 	fmt.Println(e1 == redis.Nil)

	// }
	// // if sftpPort == "" {
	// // 	sftpPort = "22" // Default SFTP port if not provided in Redis
	// // }
	// fmt.Println(s1)

	// // port, err := strconv.Atoi(sftpPort)
	// // if err != nil {
	// // 	fmt.Printf("Invalid SFTP port: %v", err)
	// // 	return
	// // }
	// // fmt.Printf("SFTP port: %d\n", port)

	//fmt.Println(extractVariables(`tm1["temperature sensor(o/p) value"] > 2 && ptm1.pressure_sensor == 'on'`))

}
