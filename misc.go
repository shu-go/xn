package main

import (
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
