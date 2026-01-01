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
	State       string `json:"state"` // pre, in, post
	Detail      string `json:"detail"`
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
	Abbreviation string `json:"abbreviation"`
}

func main() {
	// ESPN API
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

	// 输出 HTML 头部
	// 这里预定义了一些 CSS 样式，虽然目前没用到 cssClass，但保留样式表不影响编译
	fmt.Println(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body { background-color: #1c1c1e; color: #f2f2f7; font-family: "Menlo", monospace; padding: 15px; font-size: 13px; }
		h3 { color: #ff9f0a; margin: 0 0 10px 0; border-bottom: 1px solid #3a3a3c; padding-bottom: 5px; }
		pre { white-space: pre; margin: 0; }
	</style>
	</head>
	<body>
	`)

	fmt.Printf("<h3>🏀 NBA 战报 (%s)</h3>\n", currentTime)
	fmt.Println("<pre>")

	if len(result.Events) == 0 {
		fmt.Println("今天暂时没有比赛。")
	} else {
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
				// 未开始
				stateIcon = "🕒"
				// 解析 UTC 时间转本地
				t, err := time.Parse(time.RFC3339, event.Date)
				if err == nil {
					detail = t.In(time.Local).Format("15:04")
				} else {
					detail = "待定"
				}

			} else if status == "in" {
				// 进行中
				stateIcon = "🔴"
				if event.Status.DisplayClock == "0.0" {
					detail = fmt.Sprintf("Q%d 结束", event.Status.Period)
				} else {
					detail = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
				}

			} else if status == "post" {
				// 已结束
				stateIcon = "✅"
				detail = "Final"
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

	fmt.Println("</pre></body></html>")
}
