package main

import (
	"fmt"
	"os"
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

// valueFromConfigFile reports whether a value resolved by
// firstNonEmpty(argValue, fileValue, envValue) actually came from the config
// file, i.e. no higher-precedence CLI argument was given and the config file
// did hold a value.
func valueFromConfigFile(argValue, fileValue string) bool {
	return argValue == "" && fileValue != ""
}

// requireHTTPS rejects a webhook URL that does not use https, since the URL
// itself carries a secret token that must not be sent over plain HTTP.
func requireHTTPS(webhookURL string) error {
	if !strings.HasPrefix(webhookURL, "https://") {
		return fmt.Errorf("webhook URL must use https")
	}
	return nil
}

// withRetry calls fn, retrying up to retry additional times (for a total of
// retry+1 attempts) as long as it keeps returning an error. It returns the
// error from the last attempt.
func withRetry(retry int, fn func() error) error {
	err := fn()
	for i := 0; i < retry && err != nil; i++ {
		err = fn()
	}
	return err
}

// printSetEnvInstead tells the user to set environment variables rather than
// having a freshly obtained token written to the (plaintext) config file.
// It is used when the credentials used to obtain the token did not
// themselves come from the config file, so the user is deliberately keeping
// secrets out of it.
func printSetEnvInstead(pairs ...[2]string) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "set the following environment variable(s) instead:")
	for _, p := range pairs {
		fmt.Fprintf(os.Stdout, "  %s=%s\n", p[0], p[1])
	}
}
