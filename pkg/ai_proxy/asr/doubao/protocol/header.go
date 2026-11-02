package protocol

import (
	"bytes"
	"net/http"

	"github.com/google/uuid"
)

type AsrRequestHeader struct {
	messageType              MessageType
	messageTypeSpecificFlags MessageTypeSpecificFlags
	serializationType        SerializationType
	compressionType          CompressionType
	reservedData             []byte
}

func (h *AsrRequestHeader) toBytes() []byte {
	header := bytes.NewBuffer([]byte{})
	header.WriteByte(byte(PROTOCOL_VERSION<<4 | 1))
	header.WriteByte(byte(h.messageType<<4) | byte(h.messageTypeSpecificFlags))
	header.WriteByte(byte(h.serializationType<<4) | byte(h.compressionType))
	header.Write(h.reservedData)
	return header.Bytes()
}

func (h *AsrRequestHeader) WithMessageType(messageType MessageType) *AsrRequestHeader {
	h.messageType = messageType
	return h
}

func (h *AsrRequestHeader) WithMessageTypeSpecificFlags(messageTypeSpecificFlags MessageTypeSpecificFlags) *AsrRequestHeader {
	h.messageTypeSpecificFlags = messageTypeSpecificFlags
	return h
}

func (h *AsrRequestHeader) WithSerializationType(serializationType SerializationType) *AsrRequestHeader {
	h.serializationType = serializationType
	return h
}

func (h *AsrRequestHeader) WithCompressionType(compressionType CompressionType) *AsrRequestHeader {
	h.compressionType = compressionType
	return h
}

func (h *AsrRequestHeader) WithReservedData(reservedData []byte) *AsrRequestHeader {
	h.reservedData = reservedData
	return h
}

func DefaultHeader() *AsrRequestHeader {
	return &AsrRequestHeader{
		messageType:              CLIENT_FULL_REQUEST,
		messageTypeSpecificFlags: POS_SEQUENCE,
		serializationType:        JSON,
		compressionType:          GZIP,
		reservedData:             []byte{0x00},
	}
}

func NewAuthHeader(config Config) http.Header {
	header := http.Header{}
	header.Set("X-Api-Resource-Id", config.ResourceID)
	header.Set("X-Api-Request-Id", uuid.NewString())
	header.Set("X-Api-Key", config.APIKey)
	return header
}
