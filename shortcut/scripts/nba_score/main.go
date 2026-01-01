package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// --- 1. 数据结构 (保持不变) ---

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

// --- 2. 主程序 ---

func main() {
	// 获取数据
	url := "http://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Error fetching data:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	// --- 3. 输出 HTML (样式调整为精致小巧版) ---
	fmt.Println(`
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NBA Mini Scoreboard</title>
<style>
   /* 全局重置 */
   body {
      margin: 0;
      padding: 0;
      background-color: #000000;
      height: 100vh;
      display: flex;
      justify-content: center;
      align-items: center;
      font-family: 'Courier New', Courier, monospace; /* 等宽字体 */
      color: #ffffff;
   }

   /* 卡片容器：缩小尺寸 */
   .card {
      background-color: #121212;
      border-radius: 12px;       /* 圆角变小 */
      padding: 24px 30px;        /* 内边距大幅减小 */
      font-size: 14px;           /* 恢复正常阅读字号 */
      /* min-width 不再设得特别大，改为适中 */
      min-width: 400px;          
      
      display: flex;
      flex-direction: column;
      align-items: center;       /* 居中对齐 */
      
      box-shadow: 0 5px 20px rgba(0,0,0,0.8);
      border: 1px solid #333;
   }

   /* 列表容器 */
   .match-list {
      display: flex;
      flex-direction: column;
      gap: 12px; /* 行间距变紧凑 */
   }

   /* 单行比赛 */
   .match-row {
      display: flex;
      align-items: center;
      gap: 12px; /* 元素间距变紧凑 */
   }

   /* --- 列样式：尺寸微调 --- */
   
   /* 图标列 */
   .icon-box {
      width: 20px;
      height: 20px;
      display: flex;
      justify-content: center;
      align-items: center;
      border-radius: 4px;
      font-size: 12px;
      font-weight: bold;
   }
   
   .icon-final { background-color: #00b300; color: white; }
   .icon-live  { background-color: #cc0000; color: white; animation: pulse 2s infinite;}
   .icon-pre   { background-color: #333333; color: #aaa; }

   /* 队名列 */
   .team {
      width: 40px; /* 缩减宽度 */
      text-align: center;
      font-weight: bold;
   }

   /* 比分列 */
   .score {
      width: 100px; /* 缩减宽度 */
      text-align: center;
      font-weight: bold;
      color: #e0e0e0;
   }

   /* 状态文本列 */
   .status-text {
      color: #888;
      font-size: 0.85em; /* 稍微小一点 */
      width: 80px;       /* 缩减宽度 */
      text-align: right;
   }
   
   @keyframes pulse {
       0% { opacity: 1; }
       50% { opacity: 0.6; }
       100% { opacity: 1; }
   }
</style>
</head>
<body>
<div class="card">
    <div class="match-list">
`)

	// --- 4. 循环遍历比赛 (去掉了标题部分) ---
	if len(result.Events) == 0 {
		fmt.Println(`<div style="color:#666; text-align:center; padding:10px;">今天暂无比赛</div>`)
	} else {
		for _, event := range result.Events {
			comp := event.Competitions[0]
			state := event.Status.Type.State

			var home, away Competitor
			for _, c := range comp.Competitors {
				if c.HomeAway == "home" {
					home = c
				} else {
					away = c
				}
			}

			var iconClass, iconContent, scoreStr, statusText string

			switch state {
			case "pre": // 未开始
				iconClass = "icon-pre"
				iconContent = "🕒"
				scoreStr = "vs"
				t, err := time.Parse(time.RFC3339, event.Date)
				if err == nil {
					statusText = t.In(time.Local).Format("15:04")
				} else {
					statusText = "TBD"
				}

			case "in": // 进行中
				iconClass = "icon-live"
				iconContent = "●"
				scoreStr = fmt.Sprintf("%s - %s", away.Score, home.Score)
				if event.Status.DisplayClock == "0.0" {
					statusText = fmt.Sprintf("Q%d End", event.Status.Period)
				} else {
					statusText = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
				}

			case "post": // 结束
				iconClass = "icon-final"
				iconContent = "✓"
				scoreStr = fmt.Sprintf("%s - %s", away.Score, home.Score)
				statusText = "Final"
			}

			// 输出单行 HTML
			fmt.Printf(`
            <div class="match-row">
                <div class="icon-box %s">%s</div>
                <div class="team">%s</div>
                <div class="score">%s</div>
                <div class="team">%s</div>
                <div class="status-text">%s</div>
            </div>
            `, iconClass, iconContent, away.Team.Abbreviation, scoreStr, home.Team.Abbreviation, statusText)
		}
	}

	// 结束标签
	fmt.Println(`
    </div>
</div>
</body>
</html>
`)
}
