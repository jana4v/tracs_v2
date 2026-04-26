package ingest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mainframe/tm-system/internal/models"
)

// ParseTMPacket parses a raw WebSocket JSON message into a normalised paramID,
// param name, processed value, and error description.
//
// Ported from old ReceiveScTm.go: param and proc_value are lowercased and
// trimmed; err_desc is lowercased for break detection.
// paramID is preserved as-is (trimmed only) because it is a numeric CDB ID.
func ParseTMPacket(msg []byte) (paramID, param, value, errDesc string, err error) {
	var pkt models.TmPacket
	if err = json.Unmarshal(msg, &pkt); err != nil {
		return "", "", "", "", fmt.Errorf("unmarshal TmPacket: %w", err)
	}

	paramID = strings.TrimSpace(pkt.ParamID)
	param = strings.ToLower(strings.TrimSpace(pkt.Param))
	value = strings.ToLower(strings.TrimSpace(pkt.ProcValue))
	errDesc = strings.ToLower(strings.TrimSpace(pkt.ErrDesc))

	return paramID, param, value, errDesc, nil
}
