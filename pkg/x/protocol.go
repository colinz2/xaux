package x

const (
	CmdStart = "start"
	CmdEnd   = "end"

	TypeRspStart      = CmdStart
	TypeRspEnd        = CmdEnd
	TypeClose         = "close" // 代表关闭会话
	TypeSentenceStart = "SentenceStart"
	TypeRecognizing   = "recognizing"
	TypeSentenceEnd   = "SentenceEnd"
)

/*
TypeRspEnd : 1. CmdEnd
TypeClose :  服务端主动关闭识别会话

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
	SessionID uint32 `json:"sessionID"`
	TaskID    string `json:"taskID"`
	UDPPort   int32  `json:"udpPort"`
	Error     Error  `json:"error,omitempty"`
}

type End struct {
	Cmd string `json:"cmd"`
}

type EndResponse struct {
	Type  string `json:"type"`
	Error Error  `json:"error,omitempty"`
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

type CloseResponse struct {
	Type            string `json:"type"`
	Error           Error  `json:"error,omitempty"`
	ConnectionClose bool   `json:"connectionClose"`
}

type AllRequest struct {
	Cmd    string      `json:"cmd"`
	Config StartConfig `json:"config,omitempty"`
}

type AllResponse struct {
	Type            string          `json:"type"`
	SessionID       uint32          `json:"sessionID"`
	TaskID          string          `json:"taskID"`
	Error           Error           `json:"error,omitempty"`
	Result          RecognizeResult `json:"result,omitempty"`
	StartResult     StartResult     `json:"startResult,omitempty"`
	ConnectionClose bool            `json:"connectionClose"`
}
