package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	mode := flag.String("m", "", "")
	flag.Parse()

	switch *mode {
	case "score":
		runScoreboard()
	case "rank":
		runStandings()
	case "stock":
		runStock()
	default:
		fmt.Fprintf(os.Stderr, "错误: 未知模式 '%s'\n请使用: -m score 或 -m rank\n", *mode)
		os.Exit(1)
	}
}
