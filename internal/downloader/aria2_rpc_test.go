package downloader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAria2RPCClient_AddURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.addUri" {
			t.Errorf("Expected method aria2.addUri, got %s", req.Method)
		}

		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`"gid123"`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAria2RPCClient(server.URL, "secret")
	gid, err := client.AddURI("http://example.com", nil)
	if err != nil {
		t.Fatalf("AddURI failed: %v", err)
	}
	if gid != "gid123" {
		t.Errorf("Expected gid123, got %s", gid)
	}
}

func TestAria2RPCClient_TellStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		status := Aria2Status{
			GID:             "gid123",
			Status:          "active",
			TotalLength:     "1000",
			CompletedLength: "500",
			DownloadSpeed:   "100",
			Connections:     "16",
		}
		statusBytes, _ := json.Marshal(status)

		resp := RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(statusBytes),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewAria2RPCClient(server.URL, "secret")
	status, err := client.TellStatus("gid123")
	if err != nil {
		t.Fatalf("TellStatus failed: %v", err)
	}
	if status.Status != "active" {
		t.Errorf("Expected active status, got %s", status.Status)
	}
	if status.TotalLength != "1000" {
		t.Errorf("Expected totalLength 1000, got %s", status.TotalLength)
	}
}
