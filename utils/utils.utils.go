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
