package api_test

import "errors"

var errTest = errors.New("api_test: test error")

func recieveMsg(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
