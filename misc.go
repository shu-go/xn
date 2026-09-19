package main

import (
	"fmt"
	"strings"
	"time"
)

func waitForStringChan(c chan string, timeout time.Duration) string {
	select {
	case <-time.After(timeout):
		return ""
	case s := <-c:
		return s
	}
}

func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// requireHTTPS rejects a webhook URL that does not use https, since the URL
// itself carries a secret token that must not be sent over plain HTTP.
func requireHTTPS(webhookURL string) error {
	if !strings.HasPrefix(webhookURL, "https://") {
		return fmt.Errorf("webhook URL must use https")
	}
	return nil
}
