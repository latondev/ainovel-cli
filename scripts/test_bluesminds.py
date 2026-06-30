#!/usr/bin/env python3
import argparse
import json
import os
import ssl
import sys
import urllib.error
import urllib.request


def load_json(path):
    with open(path, "r", encoding="utf-8-sig") as f:
        return json.load(f)


def request_json(method, url, api_key, body=None, timeout=60):
    data = None
    headers = {"Authorization": f"Bearer {api_key}"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout, context=ssl.create_default_context()) as resp:
            text = resp.read().decode("utf-8", errors="replace")
            if not text:
                return resp.status, None
            try:
                return resp.status, json.loads(text)
            except json.JSONDecodeError:
                return resp.status, text
    except urllib.error.HTTPError as e:
        text = e.read().decode("utf-8", errors="replace")
        return e.code, text
    except Exception as e:
        return None, repr(e)


def chat_body(model, stream):
    body = {
        "model": model,
        "messages": [{"role": "user", "content": "Reply with OK only."}],
        "stream": stream,
    }
    if model.lower().startswith("gpt-5"):
        body["max_completion_tokens"] = 8
    else:
        body["max_tokens"] = 8
    return body


def print_result(status, payload):
    if status and 200 <= status < 300:
        print(f"OK HTTP {status}")
    elif status:
        print(f"FAILED HTTP {status}")
    else:
        print("FAILED")

    if payload is None:
        return
    if isinstance(payload, (dict, list)):
        text = json.dumps(payload, ensure_ascii=False, indent=2)
    else:
        text = str(payload)
    if len(text) > 2500:
        text = text[:2500] + "\n... truncated ..."
    print(text)


def main():
    parser = argparse.ArgumentParser(description="Test BluesMinds OpenAI-compatible settings.")
    parser.add_argument(
        "--config",
        default=os.path.join(os.path.expanduser("~"), ".ainovel", "config.json"),
        help="Path to ainovel config.json",
    )
    parser.add_argument("--model", action="append", dest="models", help="Model to test. Can be repeated.")
    parser.add_argument("--timeout", type=int, default=90, help="Request timeout seconds")
    parser.add_argument("--skip-stream", action="store_true", help="Skip stream=true chat test")
    args = parser.parse_args()

    cfg = load_json(args.config)
    provider_key = cfg.get("provider", "")
    provider = (cfg.get("providers") or {}).get(provider_key) or {}
    api_key = provider.get("api_key") or ""
    base_url = (provider.get("base_url") or "").rstrip("/")
    default_model = cfg.get("model") or ""

    print(f"Config: {args.config}")
    print(f"Provider key: {provider_key}")
    print(f"Provider type: {provider.get('type') or ''}")
    print(f"Base URL: {base_url}")
    print(f"Default model: {default_model}")
    print(f"API key present: {bool(api_key)}")

    if not api_key:
        print("ERROR: missing api_key", file=sys.stderr)
        return 2
    if not base_url:
        print("ERROR: missing base_url", file=sys.stderr)
        return 2

    print("\n== GET /models ==")
    status, payload = request_json("GET", f"{base_url}/models", api_key, timeout=args.timeout)
    print_result(status, payload)

    listed = []
    if isinstance(payload, dict):
        listed = [str(m.get("id")) for m in payload.get("data", []) if m.get("id")]

    models = args.models or []
    if not models:
        models = []
        for candidate in [default_model] + list(provider.get("models") or []) + listed:
            if candidate and candidate not in models:
                models.append(candidate)

    print("\nModels to test:")
    for model in models:
        print(f"- {model}")

    for model in models:
        print(f"\n== Chat test: {model} (stream=false) ==")
        status, payload = request_json(
            "POST",
            f"{base_url}/chat/completions",
            api_key,
            body=chat_body(model, False),
            timeout=args.timeout,
        )
        print_result(status, payload)

        if not args.skip_stream:
            print(f"\n== Chat test: {model} (stream=true) ==")
            status, payload = request_json(
                "POST",
                f"{base_url}/chat/completions",
                api_key,
                body=chat_body(model, True),
                timeout=args.timeout,
            )
            print_result(status, payload)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
