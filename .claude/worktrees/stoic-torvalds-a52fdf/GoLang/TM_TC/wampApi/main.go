package wampApi

import (
	shared "scg/shared"

	"github.com/gammazero/nexus/v3/client"
)

func InitWampAPI() {

	var wampArangoClient *client.Client = shared.GetWampConnection()
	RegisternosqlDbProcedures(wampArangoClient)
	RegisterTMApiProcedures(wampArangoClient)
}
