package main

import (
	"bot/telegram/shared"
	"testing"
)

func TestParsePlusOneFromMessage(t *testing.T) {
	testArray := [3]string{"+1 este mensaje si es cool", "ayy lmao kekerinos +1", "no vale esto merece un +1 jeje"}
	expected := [3]bool{true, false, false}

	for i, v := range testArray {
		plusMinus, _ := shared.ParsePlusMinusOneFromMessage(v)

		if plusMinus != expected[i] {
			t.Errorf("\"%s\" correct result is %t", testArray[i], expected[i])
		}
	}
}
