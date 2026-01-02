#!/usr/bin/env python3
import json
import sys
import urllib.request
import urllib.error
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor

TARGETS = [
    {"symbol": ".NDX", "name": "NDX"},
    {"symbol": ".SPX", "name": "SPX"}
]


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

    with ThreadPoolExecutor(max_workers=2) as executor:
        results = list(executor.map(fetch_single_stock, TARGETS))

    for res in results:
        if not res:
            continue

        emoji = "🔴"
        sign = ""
        if res["change"] >= 0:
            emoji = "🟢"
            sign = "+"

        price_str = format_number(res["price"])
        change_str = f"{sign}{res['change']:.2f} ({sign}{res['change_pct']:.2f}%)"

        item = {
            "title": f"{emoji} {res['name']}   {price_str}",
            "subtitle": f"Change: {change_str} | Updated: {res['time']}",
            "arg": price_str,
            "valid": True,
            "icon": {
                "path": "icons/NASDAQ 100.png" if res["name"] == "NDX" else "icons/S&P 500.png"
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
    except Exception:
        sys.exit(0)
