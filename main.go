package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/todo/run"
)

func main() {
	err := run.Main(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
