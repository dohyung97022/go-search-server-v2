package main

import (
	"fmt"
	"testing"
)

func TestMain(t *testing.T) {

	a, b, _ := getYoutubeAPIChannels("공포게임", "AIzaSyDIc53xLxBg4W6etfMhzuf9nqdbmsqsKOc")
	fmt.Printf("res = %v\n", a)
	fmt.Printf("next token = %v\n", b)
}
