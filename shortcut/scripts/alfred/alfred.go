package main

import (
	"encoding/json"
	"flag"
	"fmt"
)

func main() {
	mode := flag.String("m", "", "")
	flag.Parse()

	var feed AlfredFeed

	switch *mode {
	case "stock":
		feed = getStocks()
	default:
		feed = AlfredFeed{Items: []AlfredItem{{Title: "未知命令", Subtitle: "请使用: nba-score, nba-rank 或 stock"}}}
	}

	// 输出 JSON 给 Alfred
	output, _ := json.Marshal(feed)
	fmt.Println(string(output))
}

type AlfredFeed struct {
	Items []AlfredItem `json:"items"`
}

type AlfredItem struct {
	Title    string      `json:"title"`
	Subtitle string      `json:"subtitle"`
	Arg      string      `json:"arg"`
	Valid    bool        `json:"valid"`
	Icon     *AlfredIcon `json:"icon,omitempty"`
}

type AlfredIcon struct {
	Type string `json:"type,omitempty"` // "fileicon" 等
	Path string `json:"path"`
}
