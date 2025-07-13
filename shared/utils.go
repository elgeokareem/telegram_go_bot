package shared

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var CustomClient = &http.Client{
	Timeout: time.Second * 60, // Consider increasing this slightly too
	Transport: &http.Transport{
		MaxIdleConns:       100,              // Default is 100, 10 is a bit low if you expect concurrent requests
		IdleConnTimeout:    90 * time.Second, // Default is 90s
		DisableCompression: false,            // Enable compression for efficiency
		// DisableKeepAlives: false, // Or remove this line entirely
		MaxIdleConnsPerHost:   10, // Good to set this, often same as MaxIdleConns for single host
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second, // Timeout for reading response headers
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
