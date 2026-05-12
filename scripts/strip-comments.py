#!/usr/bin/env python3
from __future__ import annotations

import argparse
import ast
import io
import re
import sys
import tokenize
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

EXCLUDE_DIRS = {
    ".git", "node_modules", ".next", ".venv-itest", "__pycache__",
    "dist", "build", "vendor", ".pytest_cache", ".ruff_cache",
}

GO_DIRECTIVES = re.compile(
    r"^\s*//\s*(go:(build|generate|embed|noinline|nosplit)|"
    r"\+build|nolint|lint:|deadcode:|noinspection|export)"
)
PY_PRAGMA = re.compile(
    r"^\s*#\s*(noqa|type:|pyright:|pragma:|pylint:|mypy:|fmt:|isort:|nosec)"
)
TS_DIRECTIVE = re.compile(
    r"^\s*//\s*(@ts-|eslint-|prettier-|tslint:|stylelint-|biome-)"
)

def iter_files(root: Path, suffixes: set[str]):
    for p in root.rglob("*"):
        if p.suffix not in suffixes:
            continue
        if any(part in EXCLUDE_DIRS for part in p.parts):
            continue
        if not p.is_file():
            continue
        yield p

def strip_go_like(text: str, *, directive_re: re.Pattern[str]) -> str:
    out: list[str] = []
    n = len(text)
    i = 0
    in_str: str | None = None
    in_raw: bool = False
    buf: list[str] = []

    while i < n:
        ch = text[i]

        if in_str:
            buf.append(ch)
            if ch == "\\" and not in_raw and i + 1 < n:
                buf.append(text[i + 1])
                i += 2
                continue
            if ch == in_str:
                in_str = None
                in_raw = False
            i += 1
            continue

        if ch in ('"', "'", "`"):
            in_str = ch
            in_raw = ch == "`"
            buf.append(ch)
            i += 1
            continue

        if ch == "/" and i + 1 < n and text[i + 1] == "/":
            line_start = i
            j = i
            while j < n and text[j] != "\n":
                j += 1
            comment_line = text[line_start:j]

            line_begin_in_buf = len(buf)
            while line_begin_in_buf > 0 and buf[line_begin_in_buf - 1] != "\n":
                line_begin_in_buf -= 1
            line_prefix = "".join(buf[line_begin_in_buf:])

            if directive_re.match(line_prefix + comment_line):
                buf.append(comment_line)
                i = j
                continue

            if line_prefix.strip() == "":
                while buf and buf[-1] != "\n":
                    buf.pop()
                if j < n and text[j] == "\n":
                    j += 1
            i = j
            continue

        if ch == "/" and i + 1 < n and text[i + 1] == "*":
            j = i + 2
            while j < n - 1 and not (text[j] == "*" and text[j + 1] == "/"):
                j += 1
            j = min(j + 2, n)

            line_begin_in_buf = len(buf)
            while line_begin_in_buf > 0 and buf[line_begin_in_buf - 1] != "\n":
                line_begin_in_buf -= 1
            line_prefix = "".join(buf[line_begin_in_buf:])

            if line_prefix.strip() == "":
                while buf and buf[-1] != "\n":
                    buf.pop()
                if j < n and text[j] == "\n":
                    j += 1
            i = j
            continue

        buf.append(ch)
        i += 1

    return "".join(buf)

def collapse_blank_lines(text: str) -> str:
    lines = text.split("\n")
    out: list[str] = []
    blank_run = 0
    for line in lines:
        stripped = line.rstrip()
        if stripped == "":
            blank_run += 1
            if blank_run <= 1:
                out.append("")
        else:
            blank_run = 0
            out.append(stripped)
    while out and out[-1] == "":
        out.pop()
    if out:
        out.append("")
    return "\n".join(out)

def find_docstring_lines(text: str) -> set[int]:
    try:
        tree = ast.parse(text)
    except (SyntaxError, ValueError):
        return set()

    lines_to_blank: set[int] = set()

    def visit(node, *, parent_owns_docstring: bool = False) -> None:
        if isinstance(node, (ast.Module, ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            body = getattr(node, "body", [])
            if body and isinstance(body[0], ast.Expr) and isinstance(body[0].value, ast.Constant)\
                    and isinstance(body[0].value.value, str):
                expr = body[0]
                for ln in range(expr.lineno, expr.end_lineno + 1):
                    lines_to_blank.add(ln)

        for child in ast.iter_child_nodes(node):
            visit(child)

    visit(tree)
    return lines_to_blank

def function_needs_pass(text: str, dropped: set[int]) -> set[int]:
    try:
        tree = ast.parse(text)
    except (SyntaxError, ValueError):
        return set()

    needs_pass_after: dict[int, int] = {}

    def visit(node) -> None:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            body = node.body or []
            non_doc = []
            for stmt in body:
                if isinstance(stmt, ast.Expr) and isinstance(stmt.value, ast.Constant)\
                        and isinstance(stmt.value.value, str):
                    continue
                non_doc.append(stmt)
            if not non_doc and body:
                needs_pass_after[node.lineno] = node.col_offset
        for child in ast.iter_child_nodes(node):
            visit(child)

    visit(tree)
    return set(needs_pass_after.items())

def strip_python(text: str) -> str:
    docstring_lines = find_docstring_lines(text)

    if docstring_lines:
        original_lines = text.split("\n")
        for ln in docstring_lines:
            if 1 <= ln <= len(original_lines):
                original_lines[ln - 1] = ""
        text = "\n".join(original_lines)

    try:
        tree = ast.parse(text)
        for node in ast.walk(tree):
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
                if not node.body:
                    pass
                else:
                    real = [s for s in node.body
                            if not (isinstance(s, ast.Expr)
                                    and isinstance(s.value, ast.Constant)
                                    and isinstance(s.value.value, str))]
                    if not real:
                        body_indent = " " * (node.col_offset + 4)
                        lines = text.split("\n")
                        insert_at = node.lineno
                        while insert_at < len(lines) and lines[insert_at].strip() == "":
                            insert_at += 1
                        if insert_at >= len(lines) or not lines[insert_at].startswith(body_indent + "pass"):
                            lines.insert(node.lineno, body_indent + "pass")
                            text = "\n".join(lines)
    except (SyntaxError, ValueError):
        pass

    try:
        tokens = list(tokenize.tokenize(io.BytesIO(text.encode("utf-8")).readline))
    except (tokenize.TokenError, IndentationError, SyntaxError):
        return text

    keep: list[tokenize.TokenInfo] = []
    for tok in tokens:
        if tok.type == tokenize.COMMENT:
            if PY_PRAGMA.match(tok.string) or tok.string.startswith("#!"):
                keep.append(tok)
                continue
            continue
        keep.append(tok)

    try:
        rebuilt = tokenize.untokenize(keep)
        if isinstance(rebuilt, bytes):
            rebuilt = rebuilt.decode("utf-8")
        return rebuilt
    except Exception:
        return text

def strip_sql(text: str) -> str:
    out: list[str] = []
    i = 0
    n = len(text)
    in_str: str | None = None
    while i < n:
        ch = text[i]
        if in_str:
            out.append(ch)
            if ch == in_str:
                if i + 1 < n and text[i + 1] == in_str:
                    out.append(text[i + 1])
                    i += 2
                    continue
                in_str = None
            i += 1
            continue
        if ch in ("'", '"'):
            in_str = ch
            out.append(ch)
            i += 1
            continue
        if ch == "$" and i + 1 < n:
            j = i + 1
            while j < n and (text[j].isalnum() or text[j] == "_"):
                j += 1
            if j < n and text[j] == "$":
                tag = text[i:j + 1]
                close = text.find(tag, j + 1)
                if close >= 0:
                    out.append(text[i:close + len(tag)])
                    i = close + len(tag)
                    continue
        if ch == "-" and i + 1 < n and text[i + 1] == "-":
            j = i
            while j < n and text[j] != "\n":
                j += 1
            line_begin_in_buf = len(out)
            while line_begin_in_buf > 0 and out[line_begin_in_buf - 1] != "\n":
                line_begin_in_buf -= 1
            line_prefix = "".join(out[line_begin_in_buf:])
            if line_prefix.strip() == "":
                while out and out[-1] != "\n":
                    out.pop()
                if j < n and text[j] == "\n":
                    j += 1
            i = j
            continue
        if ch == "/" and i + 1 < n and text[i + 1] == "*":
            j = i + 2
            while j < n - 1 and not (text[j] == "*" and text[j + 1] == "/"):
                j += 1
            j = min(j + 2, n)
            i = j
            continue
        out.append(ch)
        i += 1
    return "".join(out)

def process_file(path: Path) -> bool:
    try:
        original = path.read_text(encoding="utf-8")
    except (UnicodeDecodeError, OSError):
        return False

    suffix = path.suffix
    try:
        if suffix == ".go":
            new = strip_go_like(original, directive_re=GO_DIRECTIVES)
        elif suffix in {".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}:
            new = strip_go_like(original, directive_re=TS_DIRECTIVE)
        elif suffix == ".py":
            new = strip_python(original)
        elif suffix == ".sql":
            new = strip_sql(original)
        else:
            return False
    except Exception as e:
        print(f"ERROR processing {path}: {e}", file=sys.stderr)
        return False

    new = collapse_blank_lines(new)

    if new != original:
        try:
            if suffix == ".py":
                ast.parse(new)
        except SyntaxError as e:
            print(f"WOULD BREAK {path}: {e}", file=sys.stderr)
            return False
        path.write_text(new, encoding="utf-8")
        return True
    return False

def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("targets", nargs="*", default=[
        "services", "internal", "cmd",
        "mcp-host", "mcp-orchestrator",
        "scripts", "tests",
        "frontend/src",
        "docker",
    ])
    args = parser.parse_args()

    suffixes = {".go", ".py", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".sql"}
    changed = 0
    scanned = 0

    for target in args.targets:
        base = (ROOT / target).resolve()
        if not base.exists():
            continue
        for path in iter_files(base, suffixes):
            scanned += 1
            if process_file(path):
                changed += 1

    print(f"\nScanned {scanned} files, changed {changed}.")
    return 0

if __name__ == "__main__":
    sys.exit(main())
