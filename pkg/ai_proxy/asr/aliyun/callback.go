package aliyun

import (
	"encoding/json"
	"fmt"
	"log"
)

func onTaskFailed(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session 1")
		return
	}

	fmt.Println("sessID:", sess.ID(), " onTaskFailed")
	fmt.Println("text:", text)
}

func onStarted(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session 2")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onStarted")
	fmt.Println("text:", text)
}

func onSentenceBegin(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session 3")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onSentenceBegin")
	fmt.Println("text:", text)
}

func onSentenceEnd(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session 4")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onSentenceEnd")
	fmt.Println("text:", text)
	rst := Result{}
	err := json.Unmarshal([]byte(text), &rst)
	if err != nil {
		panic(err)
		return
	}
	recognizeResponse := rst.convertToRecognizeResponse(false)
	buff, err := json.Marshal(&recognizeResponse)
	if err != nil {
		panic(err)
		return
	}
	sess.netRsp.Write(buff)
}

func onResultChanged(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session 5")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onResultChanged")
	fmt.Println("text:", text)
}

func onCompleted(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session 6")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onCompleted")
	fmt.Println("text:", text)
}

func onClose(param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid Session close")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onClose")
}
