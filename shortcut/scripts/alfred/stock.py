#!/usr/bin/env python3
import json
import sys
import urllib.request
import urllib.error
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor

TARGETS = [
    {"symbol": ".NDX", "name": "NDX"},
    {"symbol": ".SPX", "name": "SPX"},
    {"symbol": "@GC.1", "name": "Gold"}
]

OUNCE_TO_GRAM = 31.1034768


def get_usd_to_cny_rate():
    try:
        url = "https://api.exchangerate-api.com/v4/latest/USD"
        req = urllib.request.Request(url, headers={
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"
        })
        with urllib.request.urlopen(req, timeout=5) as response:
            data = json.loads(response.read().decode())
            return data.get("rates", {}).get("CNY")
    except Exception:
        return None


def fetch_single_stock(target):
    symbol = target["symbol"]
    name = target["name"]

    url = (
        f"https://quote.cnbc.com/quote-html-webservice/quote.htm"
        f"?partnerId=2&requestMethod=quick&exthrs=1&noform=1&fund=1"
        f"&output=json&symbols={symbol}"
    )

    try:
        req = urllib.request.Request(url, headers={
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"
        })

        with urllib.request.urlopen(req, timeout=5) as response:
            raw_data = response.read().decode()
            data = json.loads(raw_data)

        quotes = data.get("QuickQuoteResult", {}).get("QuickQuote", {})

        q = quotes[0] if isinstance(quotes, list) and len(quotes) > 0 else quotes

        if not isinstance(q, dict) or "last" not in q:
            return None

        last_str = q.get("last", "0").replace(",", "")
        change_str = q.get("change", "0").replace(",", "")
        pct_str = q.get("change_pct", "0").replace("%", "").replace(",", "")

        last = float(last_str)
        change = float(change_str)
        change_pct = float(pct_str)

        update_time = datetime.now().strftime("%H:%M")

        return {
            "name": name,
            "price": last,
            "change": change,
            "change_pct": change_pct,
            "time": update_time
        }

    except Exception:
        return None


def format_number(n):
    return "{:,.2f}".format(n)


def get_stocks():
    items = []

    # Fetch exchange rate in parallel with stock data for efficiency
    with ThreadPoolExecutor(max_workers=4) as executor:
        rate_future = executor.submit(get_usd_to_cny_rate)
        stock_futures = [executor.submit(fetch_single_stock, target) for target in TARGETS]
        
        usd_to_cny_rate = rate_future.result()
        results = [future.result() for future in stock_futures]

    for res in results:
        if not res:
            continue

        res_name = res["name"]
        res_price = res["price"]
        res_change = res["change"]

        if res["name"] == "Gold" and usd_to_cny_rate:
            res_name = "黄金(人民币/克)"
            res_price = (res["price"] / OUNCE_TO_GRAM) * usd_to_cny_rate
            res_change = (res["change"] / OUNCE_TO_GRAM) * usd_to_cny_rate

        emoji = "🔴"
        sign = ""
        if res["change"] >= 0:
            emoji = "🟢"
            sign = "+"

        price_str = format_number(res_price)
        change_str = f"{sign}{res_change:.2f} ({sign}{res['change_pct']:.2f}%)"

        icon_path = "icons/S&P 500.png"  # Default icon
        if res["name"] == "NDX":
            icon_path = "icons/NASDAQ 100.png"
        elif res["name"] == "Gold":
            icon_path = "icons/gold.png"

        item = {
            "title": f"{emoji} {res_name}   {price_str}",
            "subtitle": f"Change: {change_str} | Updated: {res['time']}",
            "arg": price_str,
            "valid": True,
            "icon": {
                "path": icon_path
            }
        }
        items.append(item)

    return {"items": items}


if __name__ == "__main__":
    try:
        data = get_stocks()
        if data["items"]:
            print(json.dumps(data, ensure_ascii=False))
        else:
            sys.exit(0)
    except Exception as e:
        # It's good practice to log the exception for debugging
        # but for Alfred, we often want to fail silently.
        # print(json.dumps({"items": [{"title": "Error", "subtitle": str(e)}]}))
        sys.exit(0)
