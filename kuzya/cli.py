import sys

from . import agent


def main() -> None:
    if len(sys.argv) < 2:
        sys.exit('usage: kuzya "запрос"')
    print(agent.run(" ".join(sys.argv[1:])))


if __name__ == "__main__":
    main()
