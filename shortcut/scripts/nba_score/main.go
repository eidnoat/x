package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	// 1. 设置 HTML 头部
	// 重点：
	// - 使用 Menlo 字体 (等宽)，利用空格对齐
	// - font-size: 11px (小字号，防止换行)
	// - white-space: pre (保留代码中的空格，实现对齐)
	fmt.Printf(`
	<html>
	<body style="font-family: 'Menlo', 'Courier New', monospace; font-size: 12px; color: #333;">
	<h3 style="margin: 0 0 10px 0; font-size: 14px;">🏀 NBA 战报 (%s)</h3>
	`, currentTime)

	if len(result.Events) == 0 {
		fmt.Println("<p>今天暂时没有比赛。</p>")
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

			// 状态处理
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
				scoreDisplay = fmt.Sprintf("%3s - %-3s", away.Score, home.Score) // 稍微格式化比分
			}

			// 2. 核心修改：使用 <div> 包裹每一行，并使用 padRight 辅助对齐
			// HTML 表格在快捷指令里容易乱，但 div 块级元素一定会换行
			// 我们手动拼接字符串，让它在等宽字体下对齐

			// 格式：图标 [客队] [比分] [主队] [详情]
			// 使用 &nbsp; (不换行空格) 来微调距离，或者直接用 string format

			lineContent := fmt.Sprintf("%s %s %s %s %s",
				stateIcon,
				padRight(away.Team.Abbreviation, 4), // 客队占4格
				padCenter(scoreDisplay, 11),         // 比分占11格居中
				padRight(home.Team.Abbreviation, 4), // 主队占4格
				detail,
			)

			// 替换空格为 HTML 不换行空格，防止网页压缩空格
			htmlContent := strings.ReplaceAll(lineContent, " ", "&nbsp;")

			// 每一行是一个 div，带有底部边框
			fmt.Printf(`<div style="margin-bottom: 6px; padding-bottom: 6px; border-bottom: 1px solid #eee;">%s</div>`, htmlContent)
		}
	}

	fmt.Println("</body></html>")
}

// 辅助函数：右补齐
func padRight(str string, length int) string {
	if len(str) >= length {
		return str
	}
	return str + strings.Repeat(" ", length-len(str))
}

// 辅助函数：居中补齐
func padCenter(str string, length int) string {
	if len(str) >= length {
		return str
	}
	padding := length - len(str)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + str + strings.Repeat(" ", right)
}
