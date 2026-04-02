package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func JSONOK(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func JSONError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": status, "error": msg})
}

// webhookSem 限制并发 webhook 调用数量，防止 goroutine 暴涨
var webhookSem = make(chan struct{}, 10)

func TriggerWebhook(url string, data map[string]any) {
	select {
	case webhookSem <- struct{}{}:
		defer func() { <-webhookSem }()
	default:
		fmt.Fprintf(os.Stderr, "webhook semaphore full, dropping callback to %s\n", url)
		return
	}

	body, _ := json.Marshal(data)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
