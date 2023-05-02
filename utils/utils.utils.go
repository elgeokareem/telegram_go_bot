package utils

import (
	"strings"
)

func ParsePlusOneFromMessage(message string) bool {
	splittedMessage := strings.Split(message, " ")
	isPlusOne := splittedMessage[0] == "+1"

	return isPlusOne
}
