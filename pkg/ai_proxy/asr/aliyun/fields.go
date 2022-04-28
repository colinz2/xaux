package aliyun

import "github.com/realzhangm/xaux/pkg/x"

type Result struct {
	Header  Header  `json:"header"`
	Payload Payload `json:"payload"`
}

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

type Payload struct {
	Index      int     `json:"index"`
	Time       int     `json:"time"`
	Result     string  `json:"result"`
	Confidence float64 `json:"confidence"`
	Words      []Words `json:"words"`
}

func (r *Result) convertToRecognizeResponse(interim bool) *x.RecognizeResponse {
	xr := &x.RecognizeResponse{Cmd: x.CmdRecognize}
	xr.Result.Interim = interim
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
