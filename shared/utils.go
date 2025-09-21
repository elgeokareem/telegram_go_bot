package shared

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var CustomClient = &http.Client{
	Timeout: time.Second * 30, // Reduced timeout for faster failure detection
	Transport: &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    false,
		MaxIdleConnsPerHost:   10,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableKeepAlives:     false, // Enable keep-alives for better performance
	},
}

func ParsePlusMinusOneFromMessage(message string) (bool, *int) {
	splittedMessage := strings.Split(message, " ")
	isPlusOne := splittedMessage[0] == "+1"
	isMinusOne := splittedMessage[0] == "-1"

	var value int
	if isPlusOne {
		value = 1
		return true, &value
	} else if isMinusOne {
		value = -1
		return true, &value
	} else {
		return false, nil
	}
}

func CreateDbString(schema string, user string, password string, host string, port string, dbName string) string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s", schema, user, password, host, port, dbName)
}

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

func Abs[T Integer](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
