package x

const (
	CmdEnd       = "end"
	CmdStart     = "start"
	CmdRecognize = "recognize"
)

type StartConfig struct {
	SampleRate    int32 `json:"sampleRate"`
	BitsPerSample int32 `json:"bitsPerSample"`
}

type Start struct {
	Cmd    string      `json:"cmd"`
	Config StartConfig `json:"config"`
}

type StartResponse struct {
	Cmd       string `json:"cmd"`
	SessionID uint32 `json:"sessionID"`
	UDPPort   int32  `json:"udpPort"`
}

type End struct {
	Cmd string `json:"cmd"`
}

type EndResponse struct {
	Cmd string `json:"cmd"`
	Msg string `json:"msg"`
}

type RecognizeResult struct {
}

type Recognize struct {
	Cmd    string          `json:"cmd"`
	Result RecognizeResult `json:"result"`
}

type AllRequest struct {
	Cmd    string      `json:"cmd"`
	Config StartConfig `json:"config,omitempty"`
}

type AllResponse struct {
	Cmd       string          `json:"cmd"`
	Msg       string          `json:"msg"`
	SessionID uint32          `json:"sessionID"`
	Result    RecognizeResult `json:"result,omitempty"`
}
