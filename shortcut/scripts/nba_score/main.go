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

	// --- 3. 输出 HTML (引入自动暗黑模式支持) ---
	fmt.Println(`
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NBA Scoreboard</title>
<style>
   /* --- 核心：定义颜色变量 (默认浅色模式) --- */
   :root {
      --bg-color: #f2f2f7;        /* 浅灰背景 */
      --card-bg: #ffffff;         /* 白色卡片 */
      --text-main: #1d1d1f;       /* 深黑文字 */
      --text-sub: #86868b;        /* 灰色副文本 */
      --border-color: #d1d1d6;    /* 浅色边框 */
      --shadow: 0 10px 30px rgba(0,0,0,0.1);
      
      /* 状态图标颜色 (浅色模式下) */
      --icon-pre-bg: #e5e5ea;
      --icon-pre-text: #8e8e93;
   }

   /* --- 核心：当系统检测到暗黑模式时，覆盖变量 --- */
   @media (prefers-color-scheme: dark) {
      :root {
         --bg-color: #000000;       /* 纯黑背景 */
         --card-bg: #1c1c1e;        /* 深灰卡片 */
         --text-main: #f5f5f7;      /* 亮白文字 */
         --text-sub: #98989d;       /* 灰色副文本 */
         --border-color: #38383a;   /* 深色边框 */
         --shadow: 0 10px 40px rgba(0,0,0,0.8);
         
         /* 状态图标颜色 (暗黑模式下) */
         --icon-pre-bg: #333333;
         --icon-pre-text: #aaaaaa;
      }
   }

   /* 全局重置 */
   body {
      margin: 0;
      padding: 0;
      background-color: var(--bg-color); /* 使用变量 */
      height: 100vh;
      display: flex;
      justify-content: center;
      align-items: center;
      font-family: 'Courier New', Courier, monospace;
      color: var(--text-main); /* 使用变量 */
      transition: background-color 0.3s, color 0.3s; /* 切换主题时的平滑过渡 */
   }

   /* 卡片容器：中等尺寸 (保持你喜欢的大小) */
   .card {
      background-color: var(--card-bg); /* 使用变量 */
      border-radius: 16px;
      padding: 40px 50px;
      font-size: 18px;
      min-width: 520px;
      
      display: flex;
      flex-direction: column;
      align-items: center;
      
      box-shadow: var(--shadow); /* 使用变量 */
      border: 1px solid var(--border-color); /* 使用变量 */
   }

   /* 列表容器 */
   .match-list {
      display: flex;
      flex-direction: column;
      gap: 16px;
   }

   /* 单行比赛 */
   .match-row {
      display: flex;
      align-items: center;
      gap: 15px;
   }

   /* --- 列样式 --- */
   
   /* 图标列 */
   .icon-box {
      width: 28px;
      height: 28px;
      display: flex;
      justify-content: center;
      align-items: center;
      border-radius: 6px;
      font-size: 16px;
      font-weight: bold;
   }
   
   /* 状态颜色 (Final 和 Live 在两种模式下通用，保持原色即可) */
   .icon-final { background-color: #00b300; color: white; }
   .icon-live  { background-color: #ff3b30; color: white; animation: pulse 2s infinite;}
   
   /* Pre 状态需要适配主题 */
   .icon-pre   { 
       background-color: var(--icon-pre-bg); 
       color: var(--icon-pre-text); 
   }

   /* 队名列 */
   .team {
      width: 60px;
      text-align: center;
      font-weight: bold;
   }

   /* 比分列 */
   .score {
      width: 140px;
      text-align: center;
      font-weight: bold;
      color: var(--text-main); /* 跟随主文字颜色 */
      letter-spacing: 1px;
   }

   /* 状态文本列 */
   .status-text {
      color: var(--text-sub); /* 使用副标题颜色 */
      font-size: 0.85em;
      width: 90px;
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

	// --- 4. 循环遍历比赛 (逻辑不变) ---
	if len(result.Events) == 0 {
		fmt.Println(`<div style="color:var(--text-sub); text-align:center; padding:10px;">今天暂无比赛</div>`)
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
