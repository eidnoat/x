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

type Response struct {
	Events []Event `json:"events"`
}

type Event struct {
	Date         string        `json:"date"`
	Status       Status        `json:"status"`
	Competitions []Competition `json:"competitions"`
}

type Status struct {
	Type         Type   `json:"type"`
	DisplayClock string `json:"displayClock"`
	Period       int    `json:"period"`
}

type Type struct {
	State string `json:"state"` // pre, in, post
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
	Abbreviation string `json:"abbreviation"`
}

func main() {
	url := "http://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	currentTime := time.Now().Format("2006-01-02")

	// --- CSS 核心修改 ---
	// 1. body 使用 Flex 布局实现垂直水平居中
	// 2. font-size 调大 (18px)
	// 3. line-height 增加 (1.5)
	fmt.Println(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		html, body {
			height: 100%;
			margin: 0;
			padding: 0;
			background-color: #1c1c1e;
		}
		body { 
			display: flex;
			justify-content: center; /* 水平居中 */
			align-items: center;     /* 垂直居中 */
			color: #f2f2f7; 
			font-family: "Menlo", "Courier New", monospace; 
		}
		/* 内容容器：包裹标题和表格，确保它们作为一个整体居中 */
		.container {
			display: flex;
			flex-direction: column;
			align-items: center;
			padding: 20px;
			background-color: #2c2c2e; /* 给卡片加一个稍微浅一点的背景色，突出层次感 */
			border-radius: 12px;       /* 圆角 */
			box-shadow: 0 4px 15px rgba(0,0,0,0.5); /* 阴影 */
		}
		h3 { 
			font-size: 24px;           /* 标题加大 */
			color: #ff9f0a; 
			margin: 0 0 20px 0; 
			border-bottom: 2px solid #3a3a3c; 
			padding-bottom: 10px; 
			width: 100%;
			text-align: center;
		}
		pre { 
			font-size: 18px;           /* 正文加大 */
			line-height: 1.6;          /* 增加行间距 */
			white-space: pre; 
			margin: 0; 
		}
	</style>
	</head>
	<body>
	<div class="container">
	`)

	fmt.Printf("<h3>🏀 NBA 战报 (%s)</h3>\n", currentTime)
	fmt.Println("<pre>")

	if len(result.Events) == 0 {
		fmt.Println("今天暂时没有比赛。")
	} else {
		// 初始化 tabwriter
		// minwidth=0, tabwidth=4 (拉宽一点间距), padding=2
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

		for _, event := range result.Events {
			comp := event.Competitions[0]
			status := event.Status.Type.State

			var home, away Competitor
			for _, c := range comp.Competitors {
				if c.HomeAway == "home" {
					home = c
				} else {
					away = c
				}
			}

			var stateIcon, detail string

			if status == "pre" {
				stateIcon = "🕒"
				t, err := time.Parse(time.RFC3339, event.Date)
				if err == nil {
					detail = t.In(time.Local).Format("15:04")
				} else {
					detail = "待定"
				}

			} else if status == "in" {
				stateIcon = "🔴"
				if event.Status.DisplayClock == "0.0" {
					detail = fmt.Sprintf("Q%d End", event.Status.Period)
				} else {
					detail = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
				}

			} else if status == "post" {
				stateIcon = "✅"
				detail = "Final"
			}

			scoreDisplay := "vs"
			if status != "pre" {
				scoreDisplay = fmt.Sprintf("%s - %s", away.Score, home.Score)
			}

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

	fmt.Println("</pre></div></body></html>")
}
