package aliyun

import (
	"encoding/json"
	"fmt"
)

// 错误返回
func onTaskFailed(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}

	rst := TaskFailed{}
	err := json.Unmarshal([]byte(text), &rst)
	if err != nil {
		panic(err)
		return
	}

	fmt.Println("sessID:", sess.ID(), " onTaskFailed")
	fmt.Println("text:", text)
}

// start cmd response return from AliYun
func onStarted(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onStarted")
	fmt.Println("text:", text)

	rst := Started{}
	err := json.Unmarshal([]byte(text), &rst)
	if err != nil {
		panic(err)
		return
	}
}

// 句子开始，有返回给客户端
func onSentenceBegin(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onSentenceBegin")
	fmt.Println("text:", text)

	rst := SentenceBegin{}
	err := json.Unmarshal([]byte(text), &rst)
	if err != nil {
		panic(err)
		return
	}
	xr := rst.convertToX()
	buff, err := json.Marshal(&xr)
	if err != nil {
		panic(err)
		return
	}
	sess.netRsp.Write(buff)
}

// 句子结束，有返回给客户端
func onSentenceEnd(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onSentenceEnd")
	fmt.Println("text:", text)
	rst := SentenceEnd{}
	err := json.Unmarshal([]byte(text), &rst)
	if err != nil {
		panic(err)
		return
	}
	recognizeResponse := rst.convertToX()
	buff, err := json.Marshal(&recognizeResponse)
	if err != nil {
		panic(err)
		return
	}
	sess.netRsp.Write(buff)
}

// 结果更新
func onResultChanged(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onResultChanged")
	fmt.Println("text:", text)

	rst := ResultChanged{}
	err := json.Unmarshal([]byte(text), &rst)
	if err != nil {
		return
	}
	recognizeResponse := rst.convertToX()
	buff, err := json.Marshal(&recognizeResponse)
	if err != nil {
		panic(err)
		return
	}
	sess.netRsp.Write(buff)
}

// stop cmd response return from AliYun
func onCompleted(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}
	fmt.Println("sessID:", sess.ID(), ", onCompleted")
	fmt.Println("text:", text)
}

// closed
func onClose(param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		panic("err")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onClose")
}
