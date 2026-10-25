"""Клиент к llama-server (OpenAI-совместимый /v1/chat/completions)."""
import json
import urllib.request

URL = "http://127.0.0.1:8080/v1/chat/completions"


def chat(messages: list[dict], tools: list[dict] | None = None) -> dict:
    body = {"messages": messages, "temperature": 0.2}
    if tools:
        body["tools"] = tools
    req = urllib.request.Request(
        URL, json.dumps(body).encode(), {"Content-Type": "application/json"}
    )
    with urllib.request.urlopen(req) as r:
        return json.load(r)["choices"][0]["message"]
