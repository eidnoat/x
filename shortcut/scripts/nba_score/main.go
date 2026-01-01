package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// ==========================================
// 1. 通用数据结构
// ==========================================

type TeamData struct {
	TeamTricode string  // e.g. LAL
	PlayoffRank int     // 排名
	Wins        int     // 胜
	Losses      int     // 负
	GamesBack   float64 // 胜场差
}

// ==========================================
// 2. ESPN API JSON 结构
// ==========================================

// --- 比分 (Scoreboard) ---
type ScoreResponse struct {
	Events []ScoreEvent `json:"events"`
}
type ScoreEvent struct {
	Date         string             `json:"date"`
	Status       ScoreStatus        `json:"status"`
	Competitions []ScoreCompetition `json:"competitions"`
}
type ScoreStatus struct {
	Type         ScoreType `json:"type"`
	DisplayClock string    `json:"displayClock"`
	Period       int       `json:"period"`
}
type ScoreType struct {
	State string `json:"state"`
}
type ScoreCompetition struct {
	Competitors []ScoreCompetitor `json:"competitors"`
}
type ScoreCompetitor struct {
	HomeAway string    `json:"homeAway"`
	Team     ScoreTeam `json:"team"`
	Score    string    `json:"score"`
}
type ScoreTeam struct {
	Abbreviation string `json:"abbreviation"`
}

// --- 排名 (Standings) ---
type ESPNStandingsResponse struct {
	Children []ESPNConference `json:"children"`
}
type ESPNConference struct {
	Name      string        `json:"name"`
	Standings ESPNStandings `json:"standings"`
}
type ESPNStandings struct {
	Entries []ESPNEntry `json:"entries"`
}
type ESPNEntry struct {
	Team  ESPNTeam   `json:"team"`
	Stats []ESPNStat `json:"stats"`
}
type ESPNTeam struct {
	Abbreviation string `json:"abbreviation"`
}
type ESPNStat struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// ==========================================
// 3. 主入口
// ==========================================

func main() {
	mode := flag.String("m", "score", "模式选择: 'score' (比分) 或 'rank' (排名)")
	flag.Parse()

	switch *mode {
	case "score":
		runScoreboard()
	case "rank":
		runStandings()
	default:
		fmt.Fprintf(os.Stderr, "错误: 未知模式 '%s'\n请使用: -m score 或 -m rank\n", *mode)
		os.Exit(1)
	}
}

// ==========================================
// 4. 模式 A: 比赛日比分 (Scoreboard)
// ==========================================

func runScoreboard() {
	url := "http://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取比分数据失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result ScoreResponse
	json.Unmarshal(body, &result)

	printScoreHTML(result)
}

func printScoreHTML(result ScoreResponse) {
	fmt.Println(`
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NBA Mini Scoreboard</title>
<style>
   :root {
      --bg-color: #f2f2f7; --card-bg: #ffffff; --text-main: #1d1d1f; --text-sub: #86868b;
      --border-color: #d1d1d6; --shadow: 0 10px 30px rgba(0,0,0,0.1);
      --icon-pre-bg: #e5e5ea; --icon-pre-text: #8e8e93;
   }
   @media (prefers-color-scheme: dark) {
      :root {
         --bg-color: #000000; --card-bg: #1c1c1e; --text-main: #f5f5f7; --text-sub: #98989d;
         --border-color: #38383a; --shadow: 0 10px 40px rgba(0,0,0,0.8);
         --icon-pre-bg: #333333; --icon-pre-text: #aaaaaa;
      }
   }
   body { margin: 0; padding: 0; background-color: var(--bg-color); height: 100vh; display: flex; justify-content: center; align-items: center; font-family: 'Courier New', Courier, monospace; color: var(--text-main); transition: background-color 0.3s; }
   .card { background-color: var(--card-bg); border-radius: 16px; padding: 40px 50px; font-size: 18px; min-width: 520px; display: flex; flex-direction: column; align-items: center; box-shadow: var(--shadow); border: 1px solid var(--border-color); }
   .match-list { display: flex; flex-direction: column; gap: 16px; }
   .match-row { display: flex; align-items: center; gap: 15px; }
   .icon-box { width: 28px; height: 28px; display: flex; justify-content: center; align-items: center; border-radius: 6px; font-size: 16px; font-weight: bold; }
   .icon-final { background-color: #00b300; color: white; }
   .icon-live  { background-color: #ff3b30; color: white; animation: pulse 2s infinite;}
   .icon-pre   { background-color: var(--icon-pre-bg); color: var(--icon-pre-text); }
   .team { width: 60px; text-align: center; font-weight: bold; }
   .score { width: 140px; text-align: center; font-weight: bold; letter-spacing: 1px; }
   .status-text { color: var(--text-sub); font-size: 0.85em; width: 90px; text-align: right; }
   @keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.6; } 100% { opacity: 1; } }
</style>
</head>
<body>
<div class="card"><div class="match-list">
`)
	if len(result.Events) == 0 {
		fmt.Println(`<div style="color:var(--text-sub); text-align:center;">今天暂无比赛</div>`)
	} else {
		for _, event := range result.Events {
			comp := event.Competitions[0]
			state := event.Status.Type.State
			var home, away ScoreCompetitor
			for _, c := range comp.Competitors {
				if c.HomeAway == "home" {
					home = c
				} else {
					away = c
				}
			}
			var iconClass, iconContent, scoreStr, statusText string
			switch state {
			case "pre":
				iconClass = "icon-pre"
				iconContent = "🕒"
				scoreStr = "vs"
				t, err := time.Parse(time.RFC3339, event.Date)
				if err == nil {
					statusText = t.In(time.Local).Format("15:04")
				} else {
					statusText = "TBD"
				}
			case "in":
				iconClass = "icon-live"
				iconContent = "●"
				scoreStr = fmt.Sprintf("%s - %s", away.Score, home.Score)
				if event.Status.DisplayClock == "0.0" {
					statusText = fmt.Sprintf("Q%d End", event.Status.Period)
				} else {
					statusText = fmt.Sprintf("Q%d %s", event.Status.Period, event.Status.DisplayClock)
				}
			case "post":
				iconClass = "icon-final"
				iconContent = "✓"
				scoreStr = fmt.Sprintf("%s - %s", away.Score, home.Score)
				statusText = "Final"
			}
			fmt.Printf(`<div class="match-row"><div class="icon-box %s">%s</div><div class="team">%s</div><div class="score">%s</div><div class="team">%s</div><div class="status-text">%s</div></div>`,
				iconClass, iconContent, away.Team.Abbreviation, scoreStr, home.Team.Abbreviation, statusText)
		}
	}
	fmt.Println(`</div></div></body></html>`)
}

// ==========================================
// 5. 模式 B: 球队排名 (Standings)
// ==========================================

func runStandings() {
	url := "http://site.api.espn.com/apis/v2/sports/basketball/nba/standings"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取排名数据失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result ESPNStandingsResponse
	json.Unmarshal(body, &result)

	var westTeams, eastTeams []TeamData

	for _, child := range result.Children {
		isWest := strings.Contains(child.Name, "West")
		isEast := strings.Contains(child.Name, "East")
		if !isWest && !isEast {
			continue
		}
		for _, entry := range child.Standings.Entries {
			t := TeamData{
				TeamTricode: entry.Team.Abbreviation,
				Wins:        int(getStatValue(entry.Stats, "wins")),
				Losses:      int(getStatValue(entry.Stats, "losses")),
				GamesBack:   getStatValue(entry.Stats, "gamesBehind"),
				PlayoffRank: int(getStatValue(entry.Stats, "playoffSeed")),
			}
			if isWest {
				westTeams = append(westTeams, t)
			} else {
				eastTeams = append(eastTeams, t)
			}
		}
	}

	sortTeams(westTeams)
	sortTeams(eastTeams)

	printRankHTML(westTeams, eastTeams)
}

func getStatValue(stats []ESPNStat, name string) float64 {
	for _, s := range stats {
		if s.Name == name {
			if v, ok := s.Value.(float64); ok {
				return v
			}
		}
	}
	return 0
}

func sortTeams(teams []TeamData) {
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].PlayoffRank < teams[j].PlayoffRank
	})
}

// ------------------------------------------
// CSS 修复重点区域
// ------------------------------------------

func printRankHTML(west, east []TeamData) {
	fmt.Println(`
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>NBA Standings</title>
<style>
   :root {
      --bg-color: #f2f2f7; --card-bg: #ffffff; --text-main: #1d1d1f; --text-sub: #86868b;
      --border-color: #e5e5ea; --header-text: #1d1d1f; --rank-bg: #f2f2f7;
   }
   @media (prefers-color-scheme: dark) {
      :root {
         --bg-color: #000000; --card-bg: #1c1c1e; --text-main: #f5f5f7; --text-sub: #98989d;
         --border-color: #38383a; --header-text: #ff9f0a; --rank-bg: #2c2c2e;
      }
   }
   body { margin: 0; padding: 40px 0; background-color: var(--bg-color); font-family: 'Courier New', Courier, monospace; color: var(--text-main); display: flex; justify-content: center; min-height: 100vh; }
   .container { display: flex; gap: 30px; align-items: flex-start; flex-wrap: wrap; justify-content: center; }
   .card { background-color: var(--card-bg); border-radius: 16px; padding: 25px; box-shadow: 0 10px 30px rgba(0,0,0,0.15); border: 1px solid var(--border-color); width: 420px; }
   .card-header { font-size: 18px; font-weight: bold; color: var(--header-text); margin-bottom: 20px; text-align: center; padding-bottom: 10px; border-bottom: 2px solid var(--border-color); text-transform: uppercase; letter-spacing: 1px; }
   
   /* 表格整体布局 */
   .standings-table { 
       width: 100%; 
       border-collapse: collapse; 
       font-size: 14px; 
       table-layout: fixed; /* 关键：固定列宽 */ 
   }

   /* --- 列样式定义 --- */
   
   /* 1. 排名列 (#) */
   .col-rank {
       width: 50px;           /* 固定宽度，防止挤压第二列 */
       text-align: center;
       padding: 8px 0;
   }
   
   /* 2. 队名列 (TEAM) - 强制对齐修正 */
   .col-team {
       text-align: left;
       padding: 8px 0;
       padding-left: 20px;    /* 关键：给 th 和 td 统一的左间距 */
       width: 120px;          /* 给定宽度 */
   }

   /* 3. 战绩列 (W-L) */
   .col-rec {
       text-align: right;
       padding: 8px 0;
   }

   /* 4. 胜场差列 (GB) */
   .col-gb {
       text-align: right;
       padding: 8px 0;
       color: var(--text-sub);
       width: 40px;
   }

   /* 边框处理 */
   .standings-table th { 
       color: var(--text-sub); 
       font-size: 12px; 
       border-bottom: 1px solid var(--border-color); 
       padding-bottom: 10px; /* 表头特有的底部间距 */
   }
   .standings-table td { 
       border-bottom: 1px solid var(--border-color); 
   }
   .standings-table tr:last-child td { border-bottom: none; }
   
   /* 排名图标 */
   .rank-badge { display: inline-block; width: 24px; height: 24px; line-height: 24px; border-radius: 4px; background-color: var(--rank-bg); font-size: 12px; }
   .top-6 .rank-badge { background-color: var(--header-text); color: var(--bg-color); }
   .play-in .rank-badge { border: 1px solid var(--text-sub); background-color: transparent; color: var(--text-main); }
   
   .team-name { font-weight: bold; }
</style>
</head>
<body>
<div class="container">
`)
	printTable("WESTERN CONFERENCE", west)
	printTable("EASTERN CONFERENCE", east)
	fmt.Println(`</div></body></html>`)
}

func printTable(title string, teams []TeamData) {
	fmt.Printf(`
    <div class="card">
        <div class="card-header">%s</div>
        <table class="standings-table">
            <thead>
                <tr>
                    <th class="col-rank">#</th>
                    <th class="col-team">TEAM</th>
                    <th class="col-rec">W - L</th>
                    <th class="col-gb">GB</th>
                </tr>
            </thead>
            <tbody>
    `, title)

	for _, t := range teams {
		gbStr := fmt.Sprintf("%.1f", t.GamesBack)
		if t.GamesBack == 0 {
			gbStr = "-"
		}
		rowClass := ""
		if t.PlayoffRank <= 6 {
			rowClass = "top-6"
		} else if t.PlayoffRank <= 10 {
			rowClass = "play-in"
		}

		fmt.Printf(`
            <tr class="%s">
                <td class="col-rank"><span class="rank-badge">%d</span></td>
                <td class="col-team team-name">%s</td>
                <td class="col-rec">%d - %d</td>
                <td class="col-gb">%s</td>
            </tr>
        `, rowClass, t.PlayoffRank, t.TeamTricode, t.Wins, t.Losses, gbStr)
	}
	fmt.Println(`</tbody></table></div>`)
}
