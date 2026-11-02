package protocol

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

const DefaultSampleRate = 16000

type ProtocolVersion byte
type MessageType byte
type MessageTypeSpecificFlags byte
type SerializationType byte
type CompressionType byte

const (
	PROTOCOL_VERSION = ProtocolVersion(0b0001)

	// Message Type:
	CLIENT_FULL_REQUEST       = MessageType(0b0001)
	CLIENT_AUDIO_ONLY_REQUEST = MessageType(0b0010)
	SERVER_FULL_RESPONSE      = MessageType(0b1001)
	SERVER_ERROR_RESPONSE     = MessageType(0b1111)

	// Message Type Specific Flags
	NO_SEQUENCE       = MessageTypeSpecificFlags(0b0000)
	POS_SEQUENCE      = MessageTypeSpecificFlags(0b0001)
	NEG_SEQUENCE      = MessageTypeSpecificFlags(0b0010)
	NEG_WITH_SEQUENCE = MessageTypeSpecificFlags(0b0011)

	// Message Serialization
	NO_SERIALIZATION = SerializationType(0b0000)
	JSON             = SerializationType(0b0001)

	// Message Compression
	GZIP = CompressionType(0b0001)
)

func GzipCompress(input []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(input); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return b.Bytes(), nil
}

func GzipDecompress(input []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	out, readErr := io.ReadAll(r)
	closeErr := r.Close()
	if readErr != nil {
		return nil, fmt.Errorf("gzip read: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("gzip close: %w", closeErr)
	}
	return out, nil
}
