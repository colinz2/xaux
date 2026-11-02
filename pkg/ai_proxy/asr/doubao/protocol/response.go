package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

type Word struct {
	EndTime   int    `json:"end_time"`
	StartTime int    `json:"start_time"`
	Text      string `json:"text"`
}

type Utterance struct {
	Definite  bool   `json:"definite"`
	EndTime   int    `json:"end_time"`
	StartTime int    `json:"start_time"`
	Text      string `json:"text"`
	Words     []Word `json:"words"`
}

type AsrResponsePayload struct {
	AudioInfo struct {
		Duration int `json:"duration"`
	} `json:"audio_info"`
	Result struct {
		Text      string `json:"text"`
		Additions struct {
			LogID string `json:"log_id"`
		} `json:"additions"`
		Utterances []Utterance `json:"utterances,omitempty"`
	} `json:"result"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

type AsrResponse struct {
	Code            int                 `json:"code"`
	Event           int                 `json:"event"`
	IsLastPackage   bool                `json:"is_last_package"`
	PayloadSequence int32               `json:"payload_sequence"`
	PayloadSize     int                 `json:"payload_size"`
	PayloadMsg      *AsrResponsePayload `json:"payload_msg"`
}

func ParseResponse(msg []byte) (*AsrResponse, error) {
	if len(msg) < 4 {
		return nil, fmt.Errorf("response header is too short: %d", len(msg))
	}
	headerSize := int(msg[0]&0x0f) * 4
	if headerSize < 4 || len(msg) < headerSize {
		return nil, fmt.Errorf("invalid response header size: %d", headerSize)
	}

	result := &AsrResponse{}
	messageType := MessageType(msg[1] >> 4)
	flags := MessageTypeSpecificFlags(msg[1] & 0x0f)
	serializationMethod := SerializationType(msg[2] >> 4)
	messageCompression := CompressionType(msg[2] & 0x0f)
	payload := msg[headerSize:]

	readUint32 := func() (uint32, error) {
		if len(payload) < 4 {
			return 0, fmt.Errorf("response payload is too short")
		}
		value := binary.BigEndian.Uint32(payload[:4])
		payload = payload[4:]
		return value, nil
	}

	if flags&0x01 != 0 {
		sequence, err := readUint32()
		if err != nil {
			return nil, err
		}
		result.PayloadSequence = int32(sequence)
	}
	if flags&0x02 != 0 {
		result.IsLastPackage = true
	}
	if flags&0x04 != 0 {
		event, err := readUint32()
		if err != nil {
			return nil, err
		}
		result.Event = int(event)
	}

	switch messageType {
	case SERVER_FULL_RESPONSE:
		size, err := readUint32()
		if err != nil {
			return nil, err
		}
		result.PayloadSize = int(size)
	case SERVER_ERROR_RESPONSE:
		code, err := readUint32()
		if err != nil {
			return nil, err
		}
		result.Code = int(code)
		size, err := readUint32()
		if err != nil {
			return nil, err
		}
		result.PayloadSize = int(size)
	default:
		return nil, fmt.Errorf("unsupported response message type: %d", messageType)
	}

	if len(payload) != result.PayloadSize {
		return nil, fmt.Errorf("response payload size mismatch: header=%d actual=%d", result.PayloadSize, len(payload))
	}
	if len(payload) == 0 {
		return result, nil
	}
	switch messageCompression {
	case 0:
	case GZIP:
		var err error
		payload, err = GzipDecompress(payload)
		if err != nil {
			return nil, fmt.Errorf("decompress response: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported response compression: %d", messageCompression)
	}

	switch serializationMethod {
	case JSON:
		var asrResponse AsrResponsePayload
		if err := json.Unmarshal(payload, &asrResponse); err != nil {
			return nil, fmt.Errorf("decode response JSON: %w", err)
		}
		result.PayloadMsg = &asrResponse
	case NO_SERIALIZATION:
	default:
		return nil, fmt.Errorf("unsupported response serialization: %d", serializationMethod)
	}
	return result, nil
}
