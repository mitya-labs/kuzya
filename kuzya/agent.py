"""Цикл агента: модель -> инструмент -> результат -> модель."""
import json

from . import llm
from .tools import REGISTRY

SYSTEM = "Ты Кузя, домовой skladenOS. Работаешь офлайн, действуешь только через инструменты."
MAX_STEPS = 8


def schemas() -> list[dict]:
    return [
        {"type": "function", "function": {
            "name": t.name, "description": t.description,
            "parameters": {"type": "object", "properties": {}},
        }}
        for t in REGISTRY.values()
    ]


def confirm(name: str) -> bool:
    return input(f"Кузя хочет выполнить «{name}». Разрешить? [y/N] ").strip().lower() == "y"


def run(prompt: str) -> str:
    msgs = [{"role": "system", "content": SYSTEM}, {"role": "user", "content": prompt}]
    for _ in range(MAX_STEPS):
        reply = llm.chat(msgs, schemas())
        msgs.append(reply)
        calls = reply.get("tool_calls")
        if not calls:
            return reply.get("content", "")
        for c in calls:
            name = c["function"]["name"]
            args = json.loads(c["function"].get("arguments") or "{}")
            t = REGISTRY.get(name)
            if t is None:
                out = f"нет такого инструмента: {name}"
            elif t.dangerous and not confirm(name):
                out = "пользователь отказал"
            else:
                out = t.run(**args)
            msgs.append({"role": "tool", "tool_call_id": c.get("id", ""), "content": out})
    return "Не уложился в лимит шагов."
