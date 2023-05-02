package main

import (
	"bot/telegram/utils"
	"testing"
)

func TestParsePlusOneFromMessage(t *testing.T) {
	testArray := [3]string{"+1 este mensaje si es cool", "ayy lmao kekerinos +1", "no vale esto merece un +1 jeje"}
	expected := [3]bool{true, false, false}

	for i, v := range testArray {
		result := utils.ParsePlusOneFromMessage(v)

		if result != expected[i] {
			t.Errorf("\"%s\" correct result is %t", testArray[i], expected[i])
		}
	}
}
