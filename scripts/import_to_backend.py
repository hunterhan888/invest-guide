#!/usr/bin/env python3
"""Bulk-import parsed markdown files into the InvestGuide backend via HTTP API.

Reads:  data/parsed-text/*.md
Calls:  POST /api/v1/auth/register + /auth/login + /knowledge-docs (per file)

Prerequisites:
    - Backend running (default http://localhost:8080)
    - markitdown PDFs already generated (run scripts/parse_pdfs.py first)

Usage:
    python3 scripts/import_to_backend.py
    python3 scripts/import_to_backend.py --base-url http://localhost:8080
    python3 scripts/import_to_backend.py --email admin@invest.guide --password '...'
    python3 scripts/import_to_backend.py 越南.md       # import specific files only

Environment variables (fallback for flags):
    INVESTGUIDE_BASE_URL
    INVESTGUIDE_EMAIL
    INVESTGUIDE_PASSWORD

Notes:
    - 国名从文件名提取（去 .md 后缀）；title = "对外投资合作国别指南 - <国名>"
    - 若用户已存在，登录；若不存在，注册后登录
    - 单文件失败不中断批量
"""

from __future__ import annotations

import argparse
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SRC = REPO_ROOT / "data" / "parsed-text"

DEFAULT_BASE_URL = "http://localhost:8080"
DEFAULT_EMAIL = "importer@invest.guide"


class APIError(Exception):
    def __init__(self, status: int, message: str):
        super().__init__(f"[{status}] {message}")
        self.status = status
        self.message = message


def api_call(base_url: str, method: str, path: str, token: str | None, body: dict | None) -> dict:
    url = base_url.rstrip("/") + path
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = __import__("json").dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            import json
            return json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        try:
            import json
            payload = json.loads(raw)
            msg = payload.get("error") or raw
        except Exception:
            msg = raw
        raise APIError(exc.code, msg) from exc


def ensure_user(base_url: str, email: str, password: str) -> str:
    """Register if not exists, then login; return JWT token."""
    body = {"email": email, "password": password, "displayName": "Importer"}
    try:
        resp = api_call(base_url, "POST", "/api/v1/auth/register", None, body)
        return resp["data"]["token"]
    except APIError as exc:
        if exc.status == 409:
            # 用户已存在 → 登录
            resp = api_call(base_url, "POST", "/api/v1/auth/login", None, body)
            return resp["data"]["token"]
        raise


def import_one(base_url: str, token: str, country: str, content: str) -> None:
    body = {
        "title": f"对外投资合作国别指南 - {country}",
        "country": country,
        "sourceType": "upload",
        "content": content,
    }
    api_call(base_url, "POST", "/api/v1/knowledge-docs", token, body)


def main() -> int:
    parser = argparse.ArgumentParser(description="Bulk import parsed markdown to backend.")
    parser.add_argument(
        "files", nargs="*", help="Specific .md filenames (in data/parsed-text/). Empty = all."
    )
    parser.add_argument(
        "--base-url", default=os.environ.get("INVESTGUIDE_BASE_URL", DEFAULT_BASE_URL)
    )
    parser.add_argument(
        "--email", default=os.environ.get("INVESTGUIDE_EMAIL", DEFAULT_EMAIL)
    )
    parser.add_argument(
        "--password", default=os.environ.get("INVESTGUIDE_PASSWORD"),
        help="Password for import account (required; set via --password or INVESTGUIDE_PASSWORD env)",
    )
    parser.add_argument(
        "--src-dir", default=str(DEFAULT_SRC),
        help=f"Source markdown directory (default: {DEFAULT_SRC})",
    )
    args = parser.parse_args()

    if not args.password:
        sys.stderr.write("ERROR: --password is required (or set INVESTGUIDE_PASSWORD env).\n")
        return 1

    src_dir = Path(args.src_dir)

    if args.files:
        mds = [src_dir / f for f in args.files]
        for p in mds:
            if not p.exists():
                sys.stderr.write(f"SKIP not found: {p}\n")
    else:
        mds = sorted(src_dir.glob("*.md"))

    if not mds:
        sys.stderr.write(f"No markdown files found in {src_dir}\n")
        sys.stderr.write("Run scripts/parse_pdfs.py first.\n")
        return 1

    try:
        token = ensure_user(args.base_url, args.email, args.password)
    except APIError as exc:
        sys.stderr.write(f"Auth failed: {exc}\n")
        return 1

    imported = 0
    errors: list[tuple[str, str]] = []
    for md_path in mds:
        country = md_path.stem
        content = md_path.read_text(encoding="utf-8")
        try:
            import_one(args.base_url, token, country, content)
            imported += 1
            print(f"OK   {md_path.name}")
        except APIError as exc:
            errors.append((md_path.name, str(exc)))
            print(f"FAIL {md_path.name}: {exc}")

    print()
    print(f"Imported: {imported}  Failed: {len(errors)}")
    return 0 if not errors else 2


if __name__ == "__main__":
    sys.exit(main())
