#!/usr/bin/env python3
import sys


def main(argv: list[str]) -> None:
    print(">>> 运行 Python 示例脚本")
    if argv:
        print("参数:", " ".join(argv))
    else:
        print("未收到额外参数")


if __name__ == "__main__":
    main(sys.argv[1:])
