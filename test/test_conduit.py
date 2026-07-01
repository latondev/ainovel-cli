#!/usr/bin/env python3
"""Test script for Conduit API connection."""

import json
import urllib.request
import urllib.error

CONFIG_PATH = "../.ainovel/config condut copy.json"
BASE_URL = "https://conduit.ozdoev.net/api/v1"

with open(CONFIG_PATH, encoding="utf-8") as f:
    cfg = json.load(f)

API_KEY = cfg["providers"]["conduit"]["api_key"]
MODELS = cfg["providers"]["conduit"]["models"]

HEADERS = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json",
}


def request(endpoint: str, body: dict) -> dict:
    data = json.dumps(body).encode()
    req = urllib.request.Request(f"{BASE_URL}{endpoint}", data=data, headers=HEADERS, method="POST")
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read())


def test_model(model: str) -> None:
    print(f"  [{model}] ...", end=" ", flush=True)
    payload = {
        "model": model,
        "messages": [{"role": "user", "content": "Reply with exactly: OK"}],
        "max_tokens": 16,
        "temperature": 0,
    }
    try:
        resp = request("/chat/completions", payload)
        reply = resp["choices"][0]["message"]["content"].strip()
        tokens = resp.get("usage", {}).get("total_tokens", "?")
        print(f"OK — reply: {reply!r}  tokens: {tokens}")
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        print(f"FAIL {e.code} — {body[:120]}")
    except Exception as e:
        print(f"ERROR — {e}")


def test_models_list() -> None:
    print("\n[1] GET /models")
    req = urllib.request.Request(
        f"{BASE_URL}/models",
        headers=HEADERS,
        method="GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read())
            ids = [m["id"] for m in data.get("data", [])]
            print(f"  Available models ({len(ids)}): {ids[:8]}{'...' if len(ids) > 8 else ''}")
    except urllib.error.HTTPError as e:
        print(f"  FAIL {e.code} — {e.read().decode(errors='replace')[:120]}")


def main() -> None:
    print("=== Conduit API Test ===")
    print(f"Base URL : {BASE_URL}")
    print(f"Key      : {API_KEY[:20]}...")

    test_models_list()

    print(f"\n[2] Chat completions — testing {len(MODELS)} model(s) from config:")
    for model in MODELS:
        test_model(model)

    print("\nDone.")


if __name__ == "__main__":
    main()
