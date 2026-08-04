#!/usr/bin/env python3
"""Batch-convert country guide PDFs to markdown using markitdown.

Reads:  data/original-pdf/*.pdf
Writes: data/parsed-text/<basename>.md

Usage:
    python3 scripts/parse_pdfs.py            # process all PDFs (skip existing)
    python3 scripts/parse_pdfs.py --force    # reprocess all, overwriting
    python3 scripts/parse_pdfs.py 越南.pdf   # process specific files only

Requirements:
    pip install markitdown[pdf]

Notes:
    - 输出文件名 = PDF 文件名（去 .pdf 后缀）+ .md
    - 国名直接取自文件名（如 "越南.pdf" → "越南"）
    - 跳过已存在的 .md（除非 --force），便于断点续跑
    - 单文件失败不中断批量；错误汇总打印
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    from markitdown import MarkItDown
except ImportError:
    sys.stderr.write(
        "ERROR: markitdown not installed. Run: pip install 'markitdown[pdf]'\n"
    )
    sys.exit(1)


REPO_ROOT = Path(__file__).resolve().parent.parent
DEFAULT_SRC = REPO_ROOT / "data" / "original-pdf"
DEFAULT_DST = REPO_ROOT / "data" / "parsed-text"


def convert_one(md: MarkItDown, pdf_path: Path, md_path: Path) -> str | None:
    """Convert a single PDF; return error message or None on success."""
    try:
        result = md.convert(str(pdf_path))
        md_path.parent.mkdir(parents=True, exist_ok=True)
        md_path.write_text(result.text_content, encoding="utf-8")
        return None
    except Exception as exc:  # noqa: BLE001 — 批处理需捕获所有错误继续
        return f"{exc.__class__.__name__}: {exc}"


def main() -> int:
    parser = argparse.ArgumentParser(description="Convert PDFs to markdown via markitdown.")
    parser.add_argument(
        "files", nargs="*", help="Specific PDF filenames (in data/original-pdf/). Empty = all."
    )
    parser.add_argument(
        "--force", action="store_true", help="Overwrite existing .md files."
    )
    parser.add_argument(
        "--src-dir", default=str(DEFAULT_SRC),
        help=f"Source PDF directory (default: {DEFAULT_SRC})",
    )
    parser.add_argument(
        "--dst-dir", default=str(DEFAULT_DST),
        help=f"Destination markdown directory (default: {DEFAULT_DST})",
    )
    args = parser.parse_args()

    src_dir = Path(args.src_dir)
    dst_dir = Path(args.dst_dir)

    if args.files:
        pdfs = [src_dir / f for f in args.files]
        for p in pdfs:
            if not p.exists():
                sys.stderr.write(f"SKIP not found: {p}\n")
    else:
        pdfs = sorted(src_dir.glob("*.pdf"))

    if not pdfs:
        sys.stderr.write(f"No PDFs found in {src_dir}\n")
        return 1

    md = MarkItDown()
    errors: list[tuple[str, str]] = []
    converted = 0
    skipped = 0

    for pdf in pdfs:
        md_path = dst_dir / (pdf.stem + ".md")
        if md_path.exists() and not args.force:
            skipped += 1
            continue
        err = convert_one(md, pdf, md_path)
        if err is None:
            converted += 1
            try:
                rel = md_path.relative_to(REPO_ROOT)
            except ValueError:
                rel = md_path
            print(f"OK   {pdf.name} → {rel}")
        else:
            errors.append((pdf.name, err))
            print(f"FAIL {pdf.name}: {err}")

    print()
    print(f"Converted: {converted}  Skipped: {skipped}  Failed: {len(errors)}")
    return 0 if not errors else 2


if __name__ == "__main__":
    sys.exit(main())
