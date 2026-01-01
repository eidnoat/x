package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

// 定义 ESPN API 的 JSON 结构（只提取需要的字段）
type Response struct {
	Events []Event `json:"events"`
}

type Event struct {
	ShortName    string        `json:"shortName"`
	Status       Status        `json:"status"`
	Competitions []Competition `json:"competitions"`
}

type Status struct {
	Type         Type   `json:"type"`
	DisplayClock string `json:"displayClock"`
	Period       int    `json:"period"`
}

type Type struct {
	State       string `json:"state"`  // pre, in, post
	Detail      string `json:"detail"` // e.g., "Final", "10:00 PM"
	ShortDetail string `json:"shortDetail"`
}

type Competition struct {
	Competitors []Competitor `json:"competitors"`
}

type Competitor struct {
	HomeAway string `json:"homeAway"`
	Team     Team   `json:"team"`
	Score    string `json:"score"`
}

type Team struct {
	DisplayName  string `json:"displayName"`
	Abbreviation string `json:"abbreviation"`
}

func main() {
	// ESPN NBA Scoreboard API (无需 Key)
	url := "http://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("无法连接到网络: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取数据失败: %v\n", err)
		return
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("解析 JSON 失败: %v\n", err)
		return
	}

	currentTime := time.Now().Format("2006-01-02")

	// --- 核心修改：输出 HTML 头部 ---
	// 使用 Menlo 字体保证等宽，背景深色，字号适中
	fmt.Println(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body { background-color: #1c1c1e; color: #f2f2f7; font-family: "Menlo", "Courier New", monospace; padding: 20px; font-size: 14px; }
		pre { white-space: pre-wrap; word-wrap: break-word; }
		h2 { color: #ff9f0a; margin-bottom: 10px; border-bottom: 1px solid #3a3a3c; padding-bottom: 10px; }
	</style>
	</head>
	<body>
	`)

	fmt.Printf("<h2>🏀 NBA 战报 (%s)</h2>\n", currentTime)
	fmt.Println("<pre>") // 开始预格式化文本块

	if len(result.Events) == 0 {
		fmt.Println("今天暂时没有比赛。")
	} else {
		// 使用 TabWriter 进行对齐
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, event := range result.Events {
			comp := event.Competitions[0]
			status := event.Status.Type.State
			detail := event.Status.Type.ShortDetail

			var home, away Competitor
			for _, c := range comp.Competitors {
				if c.HomeAway == "home" {
					home = c
				} else {
					away = c
				}
			}

			stateIcon := "🕒"
			if status == "in" {
				stateIcon = "🔴"
				detail = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
			} else if status == "post" {
				stateIcon = "✅"
			}

			scoreDisplay := "vs"
			if status != "pre" {
				scoreDisplay = fmt.Sprintf("%s - %s", away.Score, home.Score)
			}

			// 输出行
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t[%s]\n",
				stateIcon,
				away.Team.Abbreviation,
				scoreDisplay,
				home.Team.Abbreviation,
				detail,
			)
		}
		w.Flush()
	}

	// --- 核心修改：输出 HTML 尾部 ---
	fmt.Println("</pre></body></html>")
}
