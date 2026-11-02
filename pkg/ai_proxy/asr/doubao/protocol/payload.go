package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

type UserMeta struct {
	Uid        string `json:"uid,omitempty"`
	Did        string `json:"did,omitempty"`
	Platform   string `json:"platform,omitempty"`
	SDKVersion string `json:"sdk_version,omitempty"`
	APPVersion string `json:"app_version,omitempty"`
}

type AudioMeta struct {
	Format  string `json:"format,omitempty"`
	Codec   string `json:"codec,omitempty"`
	Rate    int    `json:"rate,omitempty"`
	Bits    int    `json:"bits,omitempty"`
	Channel int    `json:"channel,omitempty"`
}

type CorpusMeta struct {
	BoostingTableName string `json:"boosting_table_name,omitempty"`
	CorrectTableName  string `json:"correct_table_name,omitempty"`
	Context           string `json:"context,omitempty"`
}

type RequestMeta struct {
	ModelName       string     `json:"model_name,omitempty"`
	EnableITN       bool       `json:"enable_itn,omitempty"`
	EnablePUNC      bool       `json:"enable_punc,omitempty"`
	EnableDDC       bool       `json:"enable_ddc,omitempty"`
	ShowUtterances  bool       `json:"show_utterances"`
	EnableNonstream bool       `json:"enable_nonstream"`
	ResultType      string     `json:"result_type,omitempty"`
	EndWindowSize   int        `json:"end_window_size,omitempty"`
	Corpus          CorpusMeta `json:"corpus,omitempty"`
}

type AsrRequestPayload struct {
	User    UserMeta    `json:"user"`
	Audio   AudioMeta   `json:"audio"`
	Request RequestMeta `json:"request"`
}

func NewFullClientRequest(payload AsrRequestPayload) ([]byte, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal full request: %w", err)
	}
	return newClientRequest(CLIENT_FULL_REQUEST, POS_SEQUENCE, 1, payloadJSON)
}

func NewAudioOnlyRequest(seq int32, segment []byte) ([]byte, error) {
	if seq == 0 {
		return nil, fmt.Errorf("audio sequence must not be zero")
	}
	flags := POS_SEQUENCE
	if seq < 0 {
		flags = NEG_WITH_SEQUENCE
	}
	return newClientRequest(CLIENT_AUDIO_ONLY_REQUEST, flags, seq, segment)
}

func newClientRequest(messageType MessageType, flags MessageTypeSpecificFlags, seq int32, payload []byte) ([]byte, error) {
	compressed, err := GzipCompress(payload)
	if err != nil {
		return nil, err
	}

	header := DefaultHeader().WithMessageType(messageType).WithMessageTypeSpecificFlags(flags)
	request := header.toBytes()
	request = binary.BigEndian.AppendUint32(request, uint32(seq))
	request = binary.BigEndian.AppendUint32(request, uint32(len(compressed)))
	return append(request, compressed...), nil
}
