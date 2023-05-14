package utils

import (
	"fmt"
	"strings"
)

func ParsePlusOneFromMessage(message string) bool {
	splittedMessage := strings.Split(message, " ")
	isPlusOne := splittedMessage[0] == "+1"

	return isPlusOne
}

func CreateDbString(schema string, user string, password string, host string, port string, dbName string) string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s", schema, user, password, host, port, dbName)
}

func InitVariables() {

}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
