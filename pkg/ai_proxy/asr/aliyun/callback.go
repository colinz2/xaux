package aliyun

import (
	"encoding/json"
	"log"
)

func onTaskFailed(text string, param interface{}) {
	sess := callbackSession(param, "onTaskFailed", text)
	if sess != nil {
		var result TaskFailed
		decodeCallback(text, &result)
	}
}

func onStarted(text string, param interface{}) {
	sess := callbackSession(param, "onStarted", text)
	if sess != nil {
		var result Started
		decodeCallback(text, &result)
	}
}

func onSentenceBegin(text string, param interface{}) {
	sess := callbackSession(param, "onSentenceBegin", text)
	if sess == nil {
		return
	}
	var result SentenceBegin
	if decodeCallback(text, &result) {
		writeCallbackJSON(sess, result.convertToX())
	}
}

func onSentenceEnd(text string, param interface{}) {
	sess := callbackSession(param, "onSentenceEnd", text)
	if sess == nil {
		return
	}
	var result SentenceEnd
	if decodeCallback(text, &result) {
		writeCallbackJSON(sess, result.convertToX())
	}
}

func onResultChanged(text string, param interface{}) {
	sess := callbackSession(param, "onResultChanged", text)
	if sess == nil {
		return
	}
	var result ResultChanged
	if decodeCallback(text, &result) {
		writeCallbackJSON(sess, result.convertToX())
	}
}

func onCompleted(text string, param interface{}) {
	callbackSession(param, "onCompleted", text)
}

func onClose(param interface{}) {
	sess, ok := param.(*Session)
	if !ok || sess == nil {
		log.Printf("aliyun onClose: invalid session parameter")
		return
	}
	sess.nlsStatus.setClosed()
	log.Printf("sessID: %d, onClose", sess.ID())
}

func callbackSession(param interface{}, event, text string) *Session {
	sess, ok := param.(*Session)
	if !ok || sess == nil {
		log.Printf("aliyun %s: invalid session parameter", event)
		return nil
	}
	log.Printf("sessID: %d, %s, text: %s", sess.ID(), event, text)
	return sess
}

func decodeCallback(text string, target interface{}) bool {
	if err := json.Unmarshal([]byte(text), target); err != nil {
		log.Printf("decode aliyun callback: %v", err)
		return false
	}
	return true
}

func writeCallbackJSON(sess *Session, value interface{}) {
	if err := sess.writeJSON(value); err != nil {
		log.Printf("write aliyun callback for session %d: %v", sess.ID(), err)
	}
}
