package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	// 1. 输出 HTML 表格结构
	// style 说明：
	// - table: 宽度100%，无边框
	// - td: padding 增加间距，text-align 确保对齐
	fmt.Printf(`
	<html>
	<body>
	<h3>🏀 NBA 战报 (%s)</h3>
	<table border="0" cellspacing="0" cellpadding="4" style="font-family: Helvetica, sans-serif; font-size: 14px; width: 100%%;">
	`, currentTime)

	if len(result.Events) == 0 {
		fmt.Println("<tr><td>今天暂时没有比赛。</td></tr>")
	} else {
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

			// 状态逻辑
			var stateIcon, detail, colorStyle string

			// 默认颜色 (黑/白，取决于系统深色模式)
			colorStyle = ""

			if status == "pre" {
				stateIcon = "🕒"
				t, err := time.Parse(time.RFC3339, event.Date)
				if err == nil {
					detail = t.In(time.Local).Format("15:04")
				} else {
					detail = "待定"
				}
				colorStyle = "color: #888;" // 灰色

			} else if status == "in" {
				stateIcon = "🔴"
				if event.Status.DisplayClock == "0.0" {
					detail = fmt.Sprintf("Q%d End", event.Status.Period)
				} else {
					detail = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
				}
				colorStyle = "color: #FF3B30; font-weight: bold;" // 红色加粗

			} else if status == "post" {
				stateIcon = "✅"
				detail = "Final"
				colorStyle = "color: #34C759;" // 绿色
			}

			scoreDisplay := "vs"
			if status != "pre" {
				scoreDisplay = fmt.Sprintf("%s - %s", away.Score, home.Score)
			}

			// 2. 输出表格行
			// 我们使用 width 属性来稍微控制下列宽
			fmt.Printf(`
			<tr style="%s">
				<td width="20" align="center">%s</td>
				<td width="50" align="left"><b>%s</b></td>
				<td width="80" align="center">%s</td>
				<td width="50" align="right"><b>%s</b></td>
				<td align="right" style="font-size: 12px; opacity: 0.8;">%s</td>
			</tr>
			`, colorStyle, stateIcon, away.Team.Abbreviation, scoreDisplay, home.Team.Abbreviation, detail)
		}
	}

	fmt.Println("</table></body></html>")
}
