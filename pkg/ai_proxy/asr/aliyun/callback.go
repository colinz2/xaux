package aliyun

import (
	"fmt"
	"log"
)

func onTaskFailed(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), " onTaskFailed")
	fmt.Println("text:", text)
}

func onStarted(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onStarted")
	fmt.Println("text:", text)
}

func onSentenceBegin(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onSentenceBegin")
	fmt.Println("text:", text)
}

func onSentenceEnd(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onSentenceEnd")
	fmt.Println("text:", text)
}

func onResultChanged(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onResultChanged")
	fmt.Println("text:", text)
}

func onCompleted(text string, param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onCompleted")
	fmt.Println("text:", text)
}

func onClose(param interface{}) {
	sess, ok := param.(*Session)
	if !ok {
		log.Default().Fatal("invalid logger")
		return
	}

	fmt.Println("sessID:", sess.ID(), ", onClose")
}
