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
	xr := &x.SentenceStartResponse{
		Type:        x.TypeSentenceStart,
		StartResult: x.StartResult{},
	}
	xr.StartResult.Time = r.Payload.Time
	xr.StartResult.Index = r.Payload.Index
	return xr
}

func (r *ResultChanged) convertToX() *x.RecognizingResponse {
	xr := &x.RecognizingResponse{Type: x.TypeRecognizing}
	xr.Result.Interim = true
	xr.Result.Result = r.Payload.Result
	xr.Result.Confidence = r.Payload.Confidence
	xr.Result.Time = r.Payload.Time
	xr.Result.Index = r.Payload.Index
	for i := range r.Payload.Words {
		xr.Result.Words = append(xr.Result.Words, func(words Words) x.Words {
			w := x.Words{
				Text:      words.Text,
				Starttime: words.Starttime,
				Endtime:   words.Endtime,
			}
			return w
		}(r.Payload.Words[i]))
	}
	return xr
}

func (r *SentenceEnd) convertToX() *x.RecognizingResponse {
	xr := &x.RecognizingResponse{Type: x.TypeSentenceEnd}
	xr.Result.Interim = false
	xr.Result.Result = r.Payload.Result
	xr.Result.Confidence = r.Payload.Confidence
	xr.Result.Time = r.Payload.Time
	xr.Result.Index = r.Payload.Index
	for i := range r.Payload.Words {
		xr.Result.Words = append(xr.Result.Words, func(words Words) x.Words {
			w := x.Words{
				Text:      words.Text,
				Starttime: words.Starttime,
				Endtime:   words.Endtime,
			}
			return w
		}(r.Payload.Words[i]))
	}
	return xr
}
