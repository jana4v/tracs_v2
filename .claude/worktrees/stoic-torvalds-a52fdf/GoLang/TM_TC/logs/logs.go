package logs

import (
	"log"
	"os"
)

var Logger = log.New(os.Stdout, "", 0)


func saple()  {
	Logger.Println("Hello World!")


}