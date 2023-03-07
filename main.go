package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	telegramUrl := "https://api.telegram.org/bot"
	token := ""

	response, err := http.Get(telegramUrl + token + "/getMe")

	if err != nil {
		fmt.Printf("HTTP request failed with error: %s\n", err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %s\n", err)
	}

	fmt.Println(string(body))
}
