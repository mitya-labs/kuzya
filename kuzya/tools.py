"""Реестр инструментов. Модель видит только их, не shell."""
from dataclasses import dataclass
from typing import Callable


@dataclass
class Tool:
    name: str
    description: str
    run: Callable[..., str]
    dangerous: bool = False


REGISTRY: dict[str, Tool] = {}


def tool(name: str, description: str, dangerous: bool = False):
    def deco(fn: Callable[..., str]):
        REGISTRY[name] = Tool(name, description, fn, dangerous)
        return fn
    return deco
