#!/usr/bin/env python3
"""Test script for Cline API (api.cline.bot).
Docs: https://docs.cline.bot/api/models

Model format: provider/model-name
"""

import json
import urllib.request
import urllib.error

API_KEY = "sk_d25723f7b857afe85df3ed4440c6d8b996109663c0832daa85b2de6d26577b8d"
BASE_URL = "https://api.cline.bot/v1"

# All models from Cline docs (https://docs.cline.bot/api/models)
CLINE_MODELS = [
    "anthropic/claude-sonnet-4-6",    # Claude Sonnet 4.6 from Anthropic
    "openai/gpt-4o",                  # GPT-4o from OpenAI
    "google/gemini-2.5-pro",          # Gemini 2.5 Pro from Google
    "deepseek/deepseek-chat",         # DeepSeek Chat
    "minimax/minimax-m2.5",           # MiniMax M2.5
]

HEADERS = {
    "x-api-key": API_KEY,
    "Content-Type": "application/json",
    "User-Agent": "curl/8.0",
}


def post(endpoint: str, body: dict) -> dict:
    data = json.dumps(body).encode()
    req = urllib.request.Request(f"{BASE_URL}{endpoint}", data=data, headers=HEADERS, method="POST")
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read())


def test_chat(model: str, index: int, total: int) -> dict:
    """Test chat completion. Returns status dict."""
    print(f"  [{index}/{total}] {model} ...", end=" ", flush=True)
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
        return {"model": model, "status": "OK", "reply": reply, "tokens": tokens}
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        print(f"FAIL {e.code} — {body[:200]}")
        return {"model": model, "status": f"FAIL {e.code}", "error": body[:200]}
    except Exception as e:
        print(f"ERROR — {e}")
        return {"model": model, "status": "ERROR", "error": str(e)}


def main() -> None:
    print("=== Cline API Test ===")
    print(f"Base URL : {BASE_URL}")
    print(f"Auth     : x-api-key")
    print(f"Key      : {API_KEY[:20]}...")
    print(f"\nTesting {len(CLINE_MODELS)} models from docs.cline.bot/api/models:")
    for m in CLINE_MODELS:
        print(f"  - {m}")
    print()

    results = []
    for i, model in enumerate(CLINE_MODELS, 1):
        result = test_chat(model, i, len(CLINE_MODELS))
        results.append(result)

    # Summary
    print(f"\n{'='*50}")
    print(f"SUMMARY")
    print(f"{'='*50}")
    ok = [r for r in results if r["status"] == "OK"]
    fail = [r for r in results if r["status"] != "OK"]
    print(f"  Working: {len(ok)}/{len(CLINE_MODELS)}")
    for r in ok:
        print(f"    ✅ {r['model']} — reply: {r['reply']!r}")
    for r in fail:
        print(f"    ❌ {r['model']} — {r['status']}: {r.get('error','')[:120]}")
    print("\nDone.")


if __name__ == "__main__":
    main()