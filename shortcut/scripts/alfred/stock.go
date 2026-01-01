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

func getStocks() AlfredFeed {
	symbols := []struct{ Code, Name string }{
		{"^NDX", "NASDAQ 100"},
		{"^GSPC", "S&P 500"},
	}

	var items []AlfredItem
	var wg sync.WaitGroup
	results := make([]MarketData, len(symbols))

	for i, s := range symbols {
		wg.Add(1)
		go func(idx int, code, name string) {
			defer wg.Done()
			results[idx], _ = fetchYahooData(code, name)
		}(i, s.Code, s.Name)
	}
	wg.Wait()

	for _, d := range results {
		if d.Name == "" {
			continue
		}

		emoji := "🔴" // 跌
		sign := ""
		if d.Change >= 0 {
			emoji = "🟢" // 涨
			sign = "+"
		}

		// 格式: 18,345.20
		priceStr := formatComma(d.Price)
		// 格式: +120.50 (+0.65%)
		changeStr := fmt.Sprintf("%s%.2f (%s%.2f%%)", sign, d.Change, sign, d.ChangePct)

		icon := ""
		if d.Name == "NASDAQ 100" {
			icon = "./icons/NASDAQ 100.png"
		} else if d.Name == "S&P 500" {
			icon = "./icons/S&P 500.png"
		}

		items = append(items, AlfredItem{
			Title:    fmt.Sprintf("%s %s   %s", emoji, d.Name, priceStr),
			Subtitle: fmt.Sprintf("Change: %s | Updated: %s", changeStr, d.LastUpdate),
			Icon:     &AlfredIcon{Path: icon},
			Valid:    true,
			Arg:      priceStr,
		})
	}
	return AlfredFeed{Items: items}
}

func fetchJSON[T any](url string) (*T, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result T
	json.Unmarshal(body, &result)
	return &result, nil
}

func errorItem(title, subtitle string) AlfredFeed {
	return AlfredFeed{Items: []AlfredItem{{Title: title, Subtitle: subtitle, Valid: false}}}
}

// Yahoo 数据获取逻辑 (保留之前 user-agent 逻辑)
type MarketData struct {
	Name, LastUpdate         string
	Price, Change, ChangePct float64
}
type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice, ChartPreviousClose float64
				RegularMarketTime                      int64
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

func fetchYahooData(symbol, name string) (MarketData, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", symbol)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return MarketData{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var yRes YahooResponse
	json.Unmarshal(body, &yRes)

	if len(yRes.Chart.Result) == 0 {
		return MarketData{}, fmt.Errorf("no data")
	}

	meta := yRes.Chart.Result[0].Meta
	change := meta.RegularMarketPrice - meta.ChartPreviousClose
	return MarketData{
		Name: name, Price: meta.RegularMarketPrice, Change: change,
		ChangePct:  (change / meta.ChartPreviousClose) * 100,
		LastUpdate: time.Unix(meta.RegularMarketTime, 0).Format("01-02 15:04"),
	}, nil
}

func formatComma(n float64) string {
	in := int(math.Round(n * 100))
	frac := in % 100
	if frac < 0 {
		frac = -frac
	}
	in = in / 100
	s := fmt.Sprintf("%d", int(math.Abs(float64(in))))
	var out []byte
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
