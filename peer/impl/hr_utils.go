package impl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"go.dedis.ch/cs438/types"
)

func GetHMAC(req types.RequestFileMessage, key string) []byte {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		fmt.Println("GetHMAC:", "json.Marshal", err)
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(reqBytes)
	MAC := mac.Sum(nil)
	return MAC
}

func (h *HR) recvResponseFileMessage(msg *types.ResponseFileMessage) {
	h.ACMux.Lock()
	if h.A == nil || h.C == nil {
		h.A = msg.A
		h.C = msg.C
	}
	h.ACMux.Unlock()
	h.shareTable.Set(msg.I, msg.V)
}
