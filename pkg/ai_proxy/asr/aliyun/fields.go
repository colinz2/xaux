package aliyun

import "github.com/realzhangm/xaux/pkg/x"

type Header struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Status     int    `json:"status"`
	MessageID  string `json:"message_id"`
	TaskID     string `json:"task_id"`
	StatusText string `json:"status_text"`
}

type Words struct {
	Text      string `json:"text"`
	Starttime int    `json:"startTime"`
	Endtime   int    `json:"endTime"`
}

type ResultChangedPayload struct {
	Index      int     `json:"index"`
	Time       int     `json:"time"`
	Result     string  `json:"result"`
	Confidence float64 `json:"confidence"`
	Words      []Words `json:"words,omitempty"`
}

type StashResult struct {
	SentenceID  int     `json:"sentenceId"`
	BeginTime   int     `json:"beginTime"`
	Text        string  `json:"text"`
	CurrentTime int     `json:"currentTime"`
	Words       []Words `json:"words,omitempty"`
}

type SentenceEndPayload struct {
	Index          int         `json:"index"`
	Time           int         `json:"time"`
	Result         string      `json:"result"`
	Confidence     float64     `json:"confidence"`
	Words          []Words     `json:"words,omitempty"`
	Status         int         `json:"status"`
	Gender         string      `json:"gender"`
	BeginTime      int         `json:"begin_time"`
	StashResult    StashResult `json:"stash_result,omitempty"`
	AudioExtraInfo string      `json:"audio_extra_info"`
	SentenceID     string      `json:"sentence_id"`
	GenderScore    float64     `json:"gender_score"`
}

type SentenceBeginPayload struct {
	Index int `json:"index"`
	Time  int `json:"time"`
}

type Started struct {
	Header Header `json:"header"`
}

type TaskFailed struct {
	Header Header `json:"header"`
}

type SentenceBegin struct {
	Header  Header               `json:"header"`
	Payload SentenceBeginPayload `json:"payload"`
}

type ResultChanged struct {
	Header  Header               `json:"header"`
	Payload ResultChangedPayload `json:"payload"`
}

type SentenceEnd struct {
	Header  Header             `json:"header"`
	Payload SentenceEndPayload `json:"payload"`
}

func (r *SentenceBegin) convertToX() *x.SentenceStartResponse {
	return &x.SentenceStartResponse{
		Type: x.TypeSentenceStart,
		StartResult: x.StartResult{
			Index: r.Payload.Index,
			Time:  r.Payload.Time,
		},
	}
}

func (r *ResultChanged) convertToX() *x.RecognizingResponse {
	return &x.RecognizingResponse{
		Type: x.TypeRecognizing,
		Result: x.RecognizeResult{
			Interim:    true,
			Index:      r.Payload.Index,
			Time:       r.Payload.Time,
			Result:     r.Payload.Result,
			Confidence: r.Payload.Confidence,
			Words:      convertWords(r.Payload.Words),
		},
	}
}

func (r *SentenceEnd) convertToX() *x.RecognizingResponse {
	return &x.RecognizingResponse{
		Type: x.TypeSentenceEnd,
		Result: x.RecognizeResult{
			Index:      r.Payload.Index,
			Time:       r.Payload.Time,
			Result:     r.Payload.Result,
			Confidence: r.Payload.Confidence,
			Words:      convertWords(r.Payload.Words),
		},
	}
}

func convertWords(words []Words) []x.Words {
	if len(words) == 0 {
		return nil
	}
	result := make([]x.Words, len(words))
	for i, word := range words {
		result[i] = x.Words{
			Text:      word.Text,
			StartTime: word.Starttime,
			EndTime:   word.Endtime,
		}
	}
	return result
}
