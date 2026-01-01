package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

// ==========================================
// 1. 数据结构
// ==========================================

// 我们需要的精简数据模型
type MarketData struct {
	Symbol     string
	Name       string
	Price      float64
	Change     float64
	ChangePct  float64
	LastUpdate string
	IsUp       bool
}

// Yahoo Finance API 返回的复杂 JSON 结构
type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// ==========================================
// 2. 主程序
// ==========================================

func main() {
	// 定义我们要获取的指数
	// ^NDX = 纳斯达克 100
	// ^GSPC = 标普 500
	symbols := []struct {
		Code string
		Name string
	}{
		{"^NDX", "NASDAQ 100"},
		{"^GSPC", "S&P 500"},
	}

	var wg sync.WaitGroup
	results := make([]MarketData, len(symbols))

	// 并发获取数据
	for i, s := range symbols {
		wg.Add(1)
		go func(index int, code, name string) {
			defer wg.Done()
			data, err := fetchYahooData(code, name)
			if err != nil {
				// 如果失败，返回一个空结构，价格为0
				fmt.Printf("Error fetching %s: %v\n", name, err)
				results[index] = MarketData{Name: name, Symbol: code}
			} else {
				results[index] = data
			}
		}(i, s.Code, s.Name)
	}

	wg.Wait()

	// 渲染 HTML
	printHTML(results)
}

// ==========================================
// 3. 数据获取逻辑
// ==========================================

func fetchYahooData(symbol, name string) (MarketData, error) {
	// Yahoo Finance Chart API
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", symbol)

	req, _ := http.NewRequest("GET", url, nil)
	// 必须伪装 User-Agent，否则 Yahoo 会拒绝 Go 的默认请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return MarketData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return MarketData{}, fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var yRes YahooResponse
	if err := json.Unmarshal(body, &yRes); err != nil {
		return MarketData{}, err
	}

	if len(yRes.Chart.Result) == 0 {
		return MarketData{}, fmt.Errorf("no result found")
	}

	meta := yRes.Chart.Result[0].Meta
	price := meta.RegularMarketPrice
	prevClose := meta.ChartPreviousClose
	change := price - prevClose
	changePct := (change / prevClose) * 100

	// 格式化时间
	t := time.Unix(meta.RegularMarketTime, 0)
	timeStr := t.Format("15:04:05")

	return MarketData{
		Symbol:     symbol,
		Name:       name,
		Price:      price,
		Change:     change,
		ChangePct:  changePct,
		LastUpdate: timeStr,
		IsUp:       change >= 0,
	}, nil
}

// ==========================================
// 4. HTML / CSS 渲染
// ==========================================

func printHTML(data []MarketData) {
	fmt.Println(`
<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Market Watch</title>
<style>
   :root {
      --bg-color: #f2f2f7; --card-bg: #ffffff; --text-main: #1d1d1f; --text-sub: #86868b;
      --border-color: #e5e5ea; --shadow: 0 10px 30px rgba(0,0,0,0.1);
      /* 美股配色：绿涨红跌 (若习惯A股，请交换下面两个颜色代码) */
      --color-up: #34c759;   /* Green */
      --color-down: #ff3b30; /* Red */
   }
   @media (prefers-color-scheme: dark) {
      :root {
         --bg-color: #000000; --card-bg: #1c1c1e; --text-main: #f5f5f7; --text-sub: #98989d;
         --border-color: #38383a; --shadow: 0 10px 40px rgba(0,0,0,0.8);
      }
   }
   
   body { margin: 0; padding: 40px 0; background-color: var(--bg-color); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; color: var(--text-main); display: flex; justify-content: center; min-height: 100vh; align-items: center; }
   
   .container { display: flex; gap: 30px; flex-wrap: wrap; justify-content: center; }
   
   /* 卡片样式 */
   .card { 
       background-color: var(--card-bg); 
       border-radius: 20px; 
       padding: 30px; 
       box-shadow: var(--shadow); 
       border: 1px solid var(--border-color); 
       width: 300px; /* 宽度适中 */
       display: flex;
       flex-direction: column;
       align-items: center;
       text-align: center;
       transition: transform 0.2s;
   }
   
   .card:hover { transform: translateY(-5px); }

   /* 标题 (NDX / SPX) */
   .symbol-name { font-size: 16px; font-weight: 600; color: var(--text-sub); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 5px; }
   
   /* 核心价格 */
   .price { font-size: 42px; font-weight: 700; margin: 10px 0; letter-spacing: -1px; }
   
   /* 涨跌幅容器 */
   .change-box { 
       display: inline-flex; 
       align-items: center; 
       padding: 6px 16px; 
       border-radius: 20px; 
       font-weight: 600; 
       font-size: 18px;
   }
   
   /* 涨跌颜色类 */
   .trend-up { background-color: rgba(52, 199, 89, 0.15); color: var(--color-up); }
   .trend-down { background-color: rgba(255, 59, 48, 0.15); color: var(--color-down); }
   
   .timestamp { margin-top: 20px; font-size: 12px; color: var(--text-sub); }

</style>
</head>
<body>
<div class="container">
`)

	for _, d := range data {
		// 格式化数字
		priceStr := formatComma(d.Price)
		changeSign := "+"
		trendClass := "trend-up"

		if d.Change < 0 {
			changeSign = "" // 负数自带符号
			trendClass = "trend-down"
		} else if d.Change == 0 {
			changeSign = ""
			trendClass = "" // 平盘
		}

		changeStr := fmt.Sprintf("%s%.2f (%.2f%%)", changeSign, d.Change, d.ChangePct)

		fmt.Printf(`
    <div class="card">
        <div class="symbol-name">%s</div>
        <div class="price">%s</div>
        <div class="change-box %s">
            %s
        </div>
        <div class="timestamp">Update: %s</div>
    </div>
`, d.Name, priceStr, trendClass, changeStr, d.LastUpdate)
	}

	fmt.Println(`</div></body></html>`)
}

// 辅助函数：给数字加千位分隔符 (e.g. 18345.20 -> 18,345.20)
func formatComma(n float64) string {
	in := int(math.Round(n * 100))
	var out []byte

	// 处理小数部分
	frac := in % 100
	in = in / 100
	if frac < 0 {
		frac = -frac
	} // 绝对值

	// 处理整数部分
	s := fmt.Sprintf("%d", int(math.Abs(float64(in))))
	if in < 0 {
		out = append(out, '-')
	}

	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}

	return fmt.Sprintf("%s.%02d", string(out), frac)
}
