package shared

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var CustomClient = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
		DisableKeepAlives:  true,
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
