package x

const (
	CmdEnd       = "end"
	CmdStart     = "start"
	CmdRecognize = "recognize"
	CmdEvent     = "event"
	CmdError     = "error"

	EventSentenceStart = "SentenceStart"
	EventSentenceEnd   = "SentenceEnd"
)

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
	Cmd       string `json:"cmd"`
	Error     Error  `json:"error,omitempty"`
	SessionID uint32 `json:"sessionID"`
	UDPPort   int32  `json:"udpPort"`
}

type End struct {
	Cmd string `json:"cmd"`
}

type EndResponse struct {
	Cmd   string `json:"cmd"`
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

type RecognizeResponse struct {
	Cmd    string          `json:"cmd"`
	Result RecognizeResult `json:"result"`
}

type Event struct {
	Name string `json:"name"`
}

type EventResponse struct {
	Cmd   string `json:"cmd"`
	Event Event  `json:"event"`
}

type ErrorResponse struct {
	Cmd   string `json:"cmd"`
	Error Error  `json:"error,omitempty"`
}

type AllRequest struct {
	Cmd    string      `json:"cmd"`
	Config StartConfig `json:"config,omitempty"`
}

type AllResponse struct {
	Cmd       string          `json:"cmd"`
	Msg       string          `json:"msg"`
	SessionID uint32          `json:"sessionID"`
	Error     Error           `json:"error,omitempty"`
	Event     Event           `json:"event,omitempty"`
	Result    RecognizeResult `json:"result,omitempty"`
}
