#!/usr/bin/env python3
"""把文档中的占位仓库地址替换为真实地址。

用法：
    python scripts/replace-repo-urls.py <github-用户名> [仓库名覆盖...]

例如：
    python scripts/replace-repo-urls.py myname
    python scripts/replace-repo-urls.py myname new-api-backend=my-backend

默认映射：
    你的用户名/new-api-backend -> <用户名>/new-api-backend
    你的用户名/new-api-web     -> <用户名>/new-api-web

该脚本同时处理两个仓库（后端与前端同级目录）。
"""

import io
import os
import re
import sys

PLACEHOLDER = "你的用户名"

SKIP_DIRS = {".git", "node_modules", "dist", "vendor", ".next", "build"}
TARGET_SUFFIXES = (".md", ".json", ".yml", ".yaml")


def repo_root():
    return os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def find_sibling(name):
    """在父目录下寻找同级仓库。"""
    here = repo_root()
    parent = os.path.dirname(here)
    for candidate in (here, parent):
        path = os.path.join(parent, name)
        if os.path.isdir(path):
            return path
    return None


def collect_files(roots):
    files = []
    for root in roots:
        if not root:
            continue
        for dirpath, dirnames, filenames in os.walk(root):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
            for name in filenames:
                if name.endswith(TARGET_SUFFIXES):
                    files.append(os.path.join(dirpath, name))
    return files


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 1

    username = sys.argv[1]
    overrides = {}
    for arg in sys.argv[2:]:
        if "=" not in arg:
            print("忽略无效参数:", arg)
            continue
        name, value = arg.split("=", 1)
        overrides[name] = value

    backend = repo_root()
    frontend = find_sibling("new-api-web")

    mapping = {
        "new-api-backend": overrides.get("new-api-backend", username + "/new-api-backend"),
        "new-api-web": overrides.get("new-api-web", username + "/new-api-web"),
    }

    pattern = re.compile(
        r"(?:https://github\.com|git\+https://github\.com)/"
        + re.escape(PLACEHOLDER)
        + r"/([A-Za-z0-9._-]+)"
    )

    changed = []
    for path in collect_files([backend, frontend]):
        try:
            original = io.open(path, encoding="utf-8").read()
        except (OSError, UnicodeDecodeError):
            continue

        if PLACEHOLDER not in original:
            continue

        def replace(match):
            repo = match.group(1)
            target = mapping.get(repo)
            if not target:
                return match.group(0)
            prefix = match.group(0)[: match.group(0).index("https://")]
            return prefix + "https://github.com/" + target

        updated = pattern.sub(replace, original)
        if updated != original:
            io.open(path, "w", encoding="utf-8", newline="\n").write(updated)
            changed.append(os.path.relpath(path, os.path.dirname(backend)))

    if changed:
        print("已更新以下文件：")
        for path in sorted(changed):
            print("  -", path)
    else:
        print("没有找到需要替换的占位地址。")

    return 0


if __name__ == "__main__":
    sys.exit(main())
