package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// --- 1. 保持你原有的 ESPN 数据结构不变 ---

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
	State       string `json:"state"`       // pre, in, post
	Description string `json:"description"` // e.g. "Scheduled", "Halftime", "Final"
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

// --- 2. 主逻辑 ---

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

	currentTime := time.Now().Format("2006-01-02")

	// --- 3. 输出 HTML 头部 (包含大尺寸、居中样式的 CSS) ---
	fmt.Println(`
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NBA Scoreboard</title>
<style>
   /* 全局样式 */
   body {
      margin: 0;
      padding: 0;
      background-color: #000000; /* 纯黑背景 */
      height: 100vh;
      display: flex;
      justify-content: center;
      align-items: center;
      font-family: 'Courier New', Courier, monospace; /* 等宽字体 */
      color: #ffffff;
   }

   /* 卡片容器：加大尺寸 */
   .card {
      background-color: #121212;
      border-radius: 24px;
      padding: 60px 80px;  /* 巨大的内边距 */
      font-size: 22px;     /* 大字号 */
      min-width: 650px;    /* 保证卡片够宽 */
      
      display: flex;
      flex-direction: column;
      align-items: center; /* 让内部元素水平居中 */
      
      box-shadow: 0 20px 60px rgba(0,0,0,0.9);
      border: 1px solid #333;
   }

   /* 标题 */
   .header {
      font-size: 1.5em;
      font-weight: bold;
      color: #ff8c00; /* NBA 橙色 */
      margin-bottom: 40px;
      display: flex;
      align-items: center;
      gap: 15px;
   }

   /* 列表容器 */
   .match-list {
      display: flex;
      flex-direction: column;
      gap: 24px; /* 行间距 */
   }

   /* 单行比赛 */
   .match-row {
      display: flex;
      align-items: center;
      gap: 20px;
   }

   /* --- 列样式：固定宽度以保证对齐 --- */
   
   /* 图标列 */
   .icon-box {
      width: 36px;
      height: 36px;
      display: flex;
      justify-content: center;
      align-items: center;
      border-radius: 6px;
      font-size: 20px;
      font-weight: bold;
   }
   
   /* 不同的状态图标颜色 */
   .icon-final { background-color: #00b300; color: white; } /* 绿色对勾 */
   .icon-live  { background-color: #cc0000; color: white; animation: pulse 2s infinite;} /* 红色直播 */
   .icon-pre   { background-color: #333333; color: #aaa; } /* 灰色未开始 */

   /* 队名列 */
   .team {
      width: 70px;
      text-align: center;
      font-weight: bold;
      font-size: 1.1em;
   }

   /* 比分列 */
   .score {
      width: 160px;
      text-align: center;
      letter-spacing: 2px;
      font-weight: bold;
   }

   /* 状态文本列 (如 "Final", "Q4 2:00") */
   .status-text {
      color: #888;
      font-size: 0.8em;
      width: 100px;
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
`)

	// 输出标题
	fmt.Printf(`
    <div class="header">
        <span>🏀</span>
        <span>NBA 战报 (%s)</span>
    </div>
    <div class="match-list">
    `, currentTime)

	if len(result.Events) == 0 {
		fmt.Println(`<div style="color:#666; text-align:center;">今天暂时没有比赛</div>`)
	} else {
		// --- 4. 循环遍历比赛 ---
		for _, event := range result.Events {
			comp := event.Competitions[0]
			state := event.Status.Type.State // pre, in, post

			// 解析主客队
			var home, away Competitor
			for _, c := range comp.Competitors {
				if c.HomeAway == "home" {
					home = c
				} else {
					away = c
				}
			}

			// --- 5. 根据状态处理显示逻辑 ---
			var iconClass, iconContent, scoreStr, statusText string

			// 状态逻辑判断
			switch state {
			case "pre":
				// 未开始
				iconClass = "icon-pre"
				iconContent = "🕒"
				scoreStr = "vs" // 未开始显示 vs
				// 解析时间
				t, err := time.Parse(time.RFC3339, event.Date)
				if err == nil {
					statusText = t.In(time.Local).Format("15:04")
				} else {
					statusText = "待定"
				}

			case "in":
				// 进行中
				iconClass = "icon-live"
				iconContent = "●" // 圆点
				scoreStr = fmt.Sprintf("%s - %s", away.Score, home.Score)
				// 显示节数和时间
				if event.Status.DisplayClock == "0.0" {
					statusText = fmt.Sprintf("Q%d End", event.Status.Period)
				} else {
					statusText = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
				}

			case "post":
				// 已结束
				iconClass = "icon-final"
				iconContent = "✓"
				scoreStr = fmt.Sprintf("%s - %s", away.Score, home.Score)
				statusText = "Final"
			}

			// --- 6. 打印单行 HTML ---
			// 注意：这里用 fmt.Printf 拼接 HTML 字符串，不再用 tabwriter
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
    </div> </div> </body>
</html>
`)
}
