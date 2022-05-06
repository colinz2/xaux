package x

const (
	CmdStart = "start"
	CmdEnd   = "end"

	TypeStart         = CmdStart
	TypeEnd           = CmdEnd
	TypeError         = "error"
	TypeSentenceStart = "SentenceStart"
	TypeRecognizing   = "recognizing"
	TypeSentenceEnd   = "SentenceEnd"
)

/*
TypeEnd : 1. CmdEnd 2. 超时，错误等

*/

type Error struct {
	Msg string `json:"msg"`
}

type StartConfig struct {
	SampleRate    int32 `json:"sampleRate"`
	BitsPerSample int32 `json:"bitsPerSample"`
}

type Start struct {
	Cmd    string      `json:"cmd"`
	Config StartConfig `json:"config"`
}

type StartResponse struct {
	Type      string `json:"type"`
	Error     Error  `json:"error,omitempty"`
	SessionID uint32 `json:"sessionID"`
	UDPPort   int32  `json:"udpPort"`
}

type End struct {
	Cmd string `json:"cmd"`
}

type EndResponse struct {
	Type  string `json:"type"`
	Error Error  `json:"error,omitempty"`
	Msg   string `json:"msg"`
}

type Words struct {
	Text      string `json:"text"`
	Starttime int    `json:"startTime"`
	Endtime   int    `json:"endTime"`
}

type RecognizeResult struct {
	Interim    bool    `json:"interim"`
	Index      int     `json:"index"`
	Time       int     `json:"time"`
	Result     string  `json:"result"`
	Confidence float64 `json:"confidence"`
	Words      []Words `json:"words,omitempty"`
}

type StartResult struct {
	Index int `json:"index"`
	Time  int `json:"time"`
}

type SentenceStartResponse struct {
	Type        string      `json:"type"`
	StartResult StartResult `json:"startResult"`
}

type RecognizingResponse struct {
	Type   string          `json:"type"`
	Result RecognizeResult `json:"result"`
}

type SentenceEndResponse struct {
	Type   string          `json:"type"`
	Result RecognizeResult `json:"result"`
}

type ErrorResponse struct {
	Type  string `json:"type"`
	Error Error  `json:"error,omitempty"`
}

type AllRequest struct {
	Cmd    string      `json:"cmd"`
	Config StartConfig `json:"config,omitempty"`
}

type AllResponse struct {
	Type        string          `json:"type"`
	Msg         string          `json:"msg"`
	SessionID   uint32          `json:"sessionID"`
	Error       Error           `json:"error,omitempty"`
	Result      RecognizeResult `json:"result,omitempty"`
	StartResult StartResult     `json:"startResult,omitempty"`
}
