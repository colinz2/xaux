package protocol

import (
	"errors"
	"strings"
)

const (
	DefaultWSURL      = "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async"
	DefaultResourceID = "volc.seedasr.sauc.duration"
)

// Config contains the Doubao websocket endpoint and API Key authentication.
type Config struct {
	WSURL      string
	APIKey     string
	ResourceID string
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.WSURL) == "" {
		return errors.New("doubao websocket URL is empty")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return errors.New("doubao auth required: set DOUBAO_API_KEY")
	}
	if strings.TrimSpace(c.ResourceID) == "" {
		return errors.New("doubao resource ID is empty")
	}
	return nil
}
