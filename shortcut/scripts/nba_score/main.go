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

	// 生成当前日期标题
	fmt.Println("```text")
	fmt.Printf("🏀 NBA 战报 (%s)\n", time.Now().Format("2006-01-02"))
	fmt.Println("---------------------------------------")

	if len(result.Events) == 0 {
		fmt.Println("今天暂时没有比赛。")
		fmt.Println("```")
		return
	}

	// 初始化 TabWriter
	// 参数含义: output, minwidth, tabwidth, padding, padchar, flags
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

		// 使用 \t (Tab) 进行分隔，tabwriter 会自动对齐
		// 格式：状态 | 客队 | 比分 | 主队 | 详情
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t[%s]\n",
			stateIcon,
			away.Team.Abbreviation,
			scoreDisplay,
			home.Team.Abbreviation,
			detail,
		)
	}

	// 刷新缓冲区，将对齐后的内容输出
	w.Flush()

	fmt.Println("---------------------------------------")
	fmt.Println("```") // 结束 Markdown 代码块
}
