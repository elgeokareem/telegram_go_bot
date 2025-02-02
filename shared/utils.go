package shared

import (
	"fmt"
	"strings"
)

func ParsePlusMinusOneFromMessage(message string) bool {
	splittedMessage := strings.Split(message, " ")
	isPlusOne := splittedMessage[0] == "+1"
	isMinusOne := splittedMessage[0] == "-1"

	return isPlusOne || isMinusOne
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
