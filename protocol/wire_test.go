package protocol_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/qpubio/qpub-go/protocol"
)

func TestErrorInfoUnmarshalServerShape(t *testing.T) {
	var wire struct {
		Error *protocol.ErrorInfo `json:"error"`
	}
	body := []byte(`{"action":11,"error":{"code":1,"href":"h","message":"m","status_code":403}}`)
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Error == nil || wire.Error.StatusCode != 403 {
		t.Fatalf("error: %+v", wire.Error)
	}
}

func TestDataMessageUnmarshalServerShape(t *testing.T) {
	body := []byte(`{
		"action":10,
		"id":"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"timestamp":"2024-06-01T12:00:00Z",
		"channel":"room-a",
		"messages":[{"event":"e","data":{"k":"v"}}]
	}`)
	var peek struct {
		Action    protocol.ActionType `json:"action"`
		ID        string              `json:"id"`
		Timestamp time.Time           `json:"timestamp"`
		Channel   string              `json:"channel"`
	}
	if err := json.Unmarshal(body, &peek); err != nil {
		t.Fatal(err)
	}
	if peek.Action != protocol.ActionMessage || peek.Channel != "room-a" || peek.ID == "" {
		t.Fatalf("%+v", peek)
	}
}

func TestConnectionDetailsUnmarshal(t *testing.T) {
	body := []byte(`{"action":1,"connection_id":"c1","connection_details":{"alias":"a","client_id":"cl","server_id":"sv"}}`)
	var wire struct {
		Details *protocol.ConnectionDetails `json:"connection_details"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Details == nil || wire.Details.ClientID != "cl" {
		t.Fatalf("%+v", wire.Details)
	}
}
