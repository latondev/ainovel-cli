#!/usr/bin/env python3
"""Test script for api.bluesminds.com API."""

import json
import urllib.request
import urllib.error

API_KEY = "sk-isHPq7HMMBTEoOYBjSDnb1FqDJ0fgUL6E3kq2ewoD69ey5Ov"
BASE_URL = "https://api.bluesminds.com/v1"

HEADERS = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json",
    "User-Agent": "curl/8.0",
}


def get(endpoint: str) -> dict:
    req = urllib.request.Request(f"{BASE_URL}{endpoint}", headers=HEADERS, method="GET")
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read())


def post(endpoint: str, body: dict) -> dict:
    data = json.dumps(body).encode()
    req = urllib.request.Request(f"{BASE_URL}{endpoint}", data=data, headers=HEADERS, method="POST")
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read())


def test_models() -> list:
    print("\n[1] GET /models")
    resp = get("/models")
    models = [m["id"] for m in resp.get("data", [])]
    print(f"  Found {len(models)} model(s): {models}")
    return models


def test_chat(model: str) -> None:
    print(f"  [{model}] ...", end=" ", flush=True)
    try:
        resp = post("/chat/completions", {
            "model": model,
            "messages": [{"role": "user", "content": "Reply with exactly: OK"}],
            "max_tokens": 16,
            "temperature": 0,
        })
        reply = resp["choices"][0]["message"]["content"].strip()
        tokens = resp.get("usage", {}).get("total_tokens", "?")
        print(f"OK — reply: {reply!r}  tokens: {tokens}")
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        print(f"FAIL {e.code} — {body[:150]}")
    except Exception as e:
        print(f"ERROR — {e}")


def main() -> None:
    print("=== BluesMinds API Test ===")
    print(f"Base URL : {BASE_URL}")
    print(f"Key      : {API_KEY[:20]}...")

    models = test_models()

    print(f"\n[2] Chat completions — testing {len(models)} model(s):")
    for model in models:
        test_chat(model)

    print("\nDone.")


if __name__ == "__main__":
    main()
