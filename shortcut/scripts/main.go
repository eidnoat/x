package scripts

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	mode := flag.String("m", "score", "模式选择: 'score' (比分) 或 'rank' (排名)")
	flag.Parse()

	switch *mode {
	case "score":
		runScoreboard()
	case "rank":
		runStandings()
	default:
		fmt.Fprintf(os.Stderr, "错误: 未知模式 '%s'\n请使用: -m score 或 -m rank\n", *mode)
		os.Exit(1)
	}
}
