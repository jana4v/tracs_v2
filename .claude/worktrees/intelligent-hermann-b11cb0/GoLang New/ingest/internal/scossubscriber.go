package ingest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mainframe/tm-system/internal/models"
)

// ParamValue holds a single normalized parameter name and its value,
// extracted from a SCOS/SMON/ADC packet.
type ParamValue struct {
	Param string
	Value string
}

// ParseSCOSPacket parses a raw WebSocket JSON message containing a ScosPkt
// and returns the list of param+value pairs plus the error description.
// Ported from old ReceiveSmon.go: param names are lowercased and trimmed,
// values are trimmed.
func ParseSCOSPacket(msg []byte) (params []ParamValue, errDesc string, err error) {
	var pkt models.ScosPkt
	if err = json.Unmarshal(msg, &pkt); err != nil {
		return nil, "", fmt.Errorf("unmarshal ScosPkt: %w", err)
	}

	errDesc = strings.ToLower(strings.TrimSpace(pkt.Error))

	params = make([]ParamValue, 0, len(pkt.ParamList))
	for _, p := range pkt.ParamList {
		name := strings.ToLower(strings.TrimSpace(p.Mnemonic))
		val := strings.TrimSpace(p.Value)
		if name == "" {
			continue
		}
		params = append(params, ParamValue{Param: name, Value: val})
	}

	return params, errDesc, nil
}
