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
	State string `json:"state"`
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

	// --- CSS 美化区域 ---
	// 使用 Flexbox 强制垂直水平居中
	// 背景纯黑，卡片深灰，字体 Menol (等宽)
	fmt.Println(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		:root {
			--bg-color: #000000;
			--card-bg: #1c1c1e;
			--text-color: #f2f2f7;
			--accent-color: #ff9f0a;
		}
		html, body {
			height: 100%;
			margin: 0;
			padding: 0;
			background-color: var(--bg-color);
			font-family: "Menlo", "Courier New", monospace;
			display: flex;
			justify-content: center;
			align-items: center;
		}
		.card {
			background-color: var(--card-bg);
			padding: 25px;
			border-radius: 16px;
			box-shadow: 0 10px 30px rgba(0,0,0,0.5);
			min-width: 320px;
		}
		h3 {
			color: var(--accent-color);
			margin: 0 0 15px 0;
			padding-bottom: 10px;
			border-bottom: 1px solid #3a3a3c;
			font-size: 18px;
			text-align: center;
			letter-spacing: 1px;
		}
		pre {
			color: var(--text-color);
			font-size: 14px;
			line-height: 1.8; /* 增加行距，更易读 */
			white-space: pre;
			margin: 0;
		}
	</style>
	</head>
	<body>
	<div class="card">
	`)

	fmt.Printf("<h3>🏀 NBA 战报 (%s)</h3>\n", currentTime)
	fmt.Println("<pre>")

	if len(result.Events) == 0 {
		fmt.Println("今天暂时没有比赛。")
	} else {
		// 初始化对齐工具
		// minwidth=0, tabwidth=4, padding=2
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

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
				// 为了美观，把比分也对其一下
				scoreDisplay = fmt.Sprintf("%3s - %-3s", away.Score, home.Score)
			}

			// 写入 Buffer
			// 格式：状态 | 客队 | 比分 | 主队 | 详情
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
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
