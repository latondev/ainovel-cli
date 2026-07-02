"""Probe - try anthropic model names + confirm working models."""
import json, urllib.request, urllib.error

API_KEY = "sk_d25723f7b857afe85df3ed4440c6d8b996109663c0832daa85b2de6d26577b8d"
H = {"Authorization": f"Bearer {API_KEY}", "Content-Type": "application/json"}
URL = "https://api.cline.bot/api/v1/chat/completions"

# Anthropic model names to try
models = [
    "anthropic/claude-sonnet-4-6",
    "anthropic/claude-sonnet-4-20250514",
    "anthropic/claude-3-5-sonnet-20241022",
    "anthropic/claude-3-opus-20240229",
    "anthropic/claude-3-5-haiku-20241022",
    # Also try models endpoint
]

for model in models:
    body = {"model": model, "messages": [{"role": "user", "content": "Say OK"}], "max_tokens": 16}
    print(f"{model} ...", end=" ", flush=True)
    try:
        req = urllib.request.Request(URL, data=json.dumps(body).encode(), headers=H, method="POST")
        with urllib.request.urlopen(req, timeout=15) as resp:
            r = json.loads(resp.read())
            content = r.get("data", r).get("choices", [{}])[0].get("message", {}).get("content", "")
            print(f"OK — {content!r}")
    except urllib.error.HTTPError as e:
        print(f"FAIL {e.code} — {e.read().decode(errors='replace')[:150]}")
    except Exception as e:
        print(f"ERROR — {e}")

# Also get /models
print("\n--- Models endpoint ---")
try:
    req = urllib.request.Request("https://api.cline.bot/api/v1/models", headers=H)
    with urllib.request.urlopen(req, timeout=10) as resp:
        data = json.loads(resp.read())
        print(json.dumps(data, ensure_ascii=False, indent=2)[:2000])
except Exception as e:
    print(f"FAIL: {e}")