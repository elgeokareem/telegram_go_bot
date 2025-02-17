package main

import (
	"bot/telegram/shared"
	"testing"
)

func TestParsePlusOneFromMessage(t *testing.T) {
	testArray := [4]string{"+1 este mensaje si es cool", "ayy lmao kekerinos +1", "no vale esto merece un -1 jeje", "-1 esto si menos 1"}

	one := 1
	minusOne := -1
	expected := [4]struct {
		isPlusMinus bool
		value       *int
	}{
		{true, &one},
		{false, &one},
		{false, &minusOne},
		{true, &minusOne},
	}

	for i, v := range testArray {
		isPlusMinus, _ := shared.ParsePlusMinusOneFromMessage(v)

		if isPlusMinus != expected[i].isPlusMinus {
			t.Errorf("\"%s\" correct result is %t", testArray[i], expected[i].isPlusMinus)
		}
	}
}
