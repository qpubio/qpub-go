package connection

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/protocol"
)

func TestCompletePingRTT(t *testing.T) {
	c := &Conn{
		pendingPings: make(map[string]pendingPing),
	}
	start := time.Now()
	ch := make(chan time.Duration, 1)
	c.pendingPings["7"] = pendingPing{start: start, ch: ch}

	raw, _ := json.Marshal(map[string]interface{}{"action": protocol.ActionPong, "id": 7})
	var wire struct {
		Action protocol.ActionType `json:"action"`
		ID     int                 `json:"id"`
	}
	_ = json.Unmarshal(raw, &wire)
	c.completePing(wire.ID)

	select {
	case rtt := <-ch:
		if rtt < 0 {
			t.Fatalf("rtt=%v", rtt)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
