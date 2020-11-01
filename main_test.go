package main

import (
	"fmt"
	"testing"
)

func TestMain(t *testing.T) {
	res := getYoutubeAPIChannelsHandler("공포게임")
	fmt.Printf("res = %v\n", res)
}
