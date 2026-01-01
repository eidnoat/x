package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	// 生成当前日期标题
	currentTime := time.Now().Format("2006-01-02")
	fmt.Printf("🏀 NBA 战报 (%s)\n", currentTime)
	fmt.Println("--------------------------------")

	if len(result.Events) == 0 {
		fmt.Println("今天暂时没有比赛。")
		return
	}

	for _, event := range result.Events {
		comp := event.Competitions[0]
		status := event.Status.Type.State // pre, in, post
		detail := event.Status.Type.ShortDetail

		var home, away Competitor
		// 区分主客场
		for _, c := range comp.Competitors {
			if c.HomeAway == "home" {
				home = c
			} else {
				away = c
			}
		}

		// 格式化输出
		// 图标状态：🔴进行中，✅已结束，🕒未开始
		stateIcon := "🕒"
		if status == "in" {
			stateIcon = "🔴"
			detail = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
		} else if status == "post" {
			stateIcon = "✅"
		}

		// 输出格式：客队 vs 主队
		// 例如：[✅] LAL (110) - (105) GSW [Final]
		scoreDisplay := "vs"
		if status != "pre" {
			scoreDisplay = fmt.Sprintf("%s - %s", away.Score, home.Score)
		}

		fmt.Printf("%s %s %s %s  [%s]\n",
			stateIcon,
			padRight(away.Team.Abbreviation, 4),
			scoreDisplay,
			padRight(home.Team.Abbreviation, 4),
			detail,
		)
	}
	fmt.Println("--------------------------------")
}

// 辅助函数：右侧填充空格以对齐
func padRight(str string, length int) string {
	if len(str) >= length {
		return str
	}
	return str + strings.Repeat(" ", length-len(str))
}
