package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ziuus/goroute/pkg/storage"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// EventPayload represents the standard webhook body delivered to subscribers.
type EventPayload struct {
	Event     string      `json:"event"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// DispatchEvent asynchronously broadcasts an event to matching, enabled webhooks.
func DispatchEvent(event string, data interface{}) {
	go func() {
		store := storage.GetWebhookStore()
		if store == nil {
			return
		}
		webhooks, err := store.ListWebhooks()
		if err != nil || len(webhooks) == 0 {
			return
		}

		payload := EventPayload{
			Event:     event,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      data,
		}
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return
		}

		for _, wh := range webhooks {
			if !wh.Enabled {
				continue
			}
			matched := false
			for _, ev := range wh.Events {
				if ev == "*" || ev == event {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}

			// Deliver payload
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, wh.URL, bytes.NewReader(bodyBytes))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "GoRoute-Webhook-Engine/0.3.0")

			if wh.Secret != "" {
				mac := hmac.New(sha256.New, []byte(wh.Secret))
				mac.Write(bodyBytes)
				req.Header.Set("X-GoRoute-Signature", hex.EncodeToString(mac.Sum(nil)))
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				log.Printf("[Webhook] Delivery error to %s: %v", wh.URL, err)
				continue
			}
			resp.Body.Close()
		}
	}()
}
