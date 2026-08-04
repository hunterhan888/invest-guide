# scripts/ — 数据预处理与批量导入

这两个脚本属于**后端流水线之外**的外部数据预处理环节，与 `backend/` 内的 Go 代码完全解耦。

## 流程定位

```
data/original-pdf/*.pdf          ← 181 个国别指南 PDF
        │
        │  scripts/parse_pdfs.py（markitdown）
        ▼
data/parsed-text/*.md            ← 预处理产物（markdown）
        │
        │  scripts/import_to_backend.py（HTTP API）
        ▼
backend knowledge 模块入库流水线  →  检索可用
```

后端 `domain/knowledge` 的 `parser.go` 不直接处理 PDF，只接受文本/markdown/HTML 字符串。

## 环境要求

- Python 3.10+
- `pip install 'markitdown[pdf]'`
- 批量导入脚本不需要额外依赖（只用标准库 `urllib`）

## 使用

### 1. PDF → markdown

```bash
# 一次性转换全部 PDF（增量、可重复执行）
python3 scripts/parse_pdfs.py \
  --src-dir ../data/original-pdf \
  --dst-dir ../data/parsed-text

# 强制重新转换已存在的 .md
python3 scripts/parse_pdfs.py --force --src-dir ../data/original-pdf

# 只转指定文件
python3 scripts/parse_pdfs.py 越南.pdf 泰国.pdf --src-dir ../data/original-pdf
```

### 2. markdown → 后端知识库

先启动后端：

```bash
make backend-dev
```

再跑批量导入（`--password` 必填，或设置 `INVESTGUIDE_PASSWORD` 环境变量）：

```bash
python3 scripts/import_to_backend.py \
  --base-url http://localhost:8080 \
  --src-dir ../data/parsed-text \
  --password 'your-password'

# 自定义账号（脚本会注册新用户；若已存在则登录）
python3 scripts/import_to_backend.py \
  --email admin@invest.guide --password 'your-strong-password'

# 只导入指定文件
python3 scripts/import_to_backend.py 越南.md --src-dir ../data/parsed-text
```

## 路径说明

两个脚本都支持 `--src-dir` / `--dst-dir` 参数，默认指向本项目内的
`data/original-pdf/` 与 `data/parsed-text/`。本仓库当前实际数据位于上一级
`../data/` 目录，请在命令行显式传入 `--src-dir` / `--dst-dir`。

## 失败处理

两个脚本都采用"单文件失败不中断批量"策略；末尾汇总 `Converted/Failed` 或
`Imported/Failed` 计数。失败列表详见终端输出。

退出码：`0` 全部成功 / `1` 输入错误 / `2` 部分文件失败。
