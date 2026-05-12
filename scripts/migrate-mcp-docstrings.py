#!/usr/bin/env python3
from __future__ import annotations

import ast
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
TOOLS_DIR = ROOT / "mcp-orchestrator" / "tools"

def is_mcp_tool_decorator(deco: ast.expr) -> bool:
    if isinstance(deco, ast.Call):
        return is_mcp_tool_decorator(deco.func)
    if isinstance(deco, ast.Attribute):
        return deco.attr == "tool"
    if isinstance(deco, ast.Name):
        return deco.id == "tool"
    return False

def find_replacements(text: str) -> list[tuple[ast.AST, str, str]]:

    try:
        tree = ast.parse(text)
    except SyntaxError:
        return []

    repl: list[tuple[ast.AST, str, str]] = []

    def visit(node: ast.AST) -> None:
        for child in ast.iter_child_nodes(node):
            if isinstance(child, (ast.FunctionDef, ast.AsyncFunctionDef)):
                for deco in child.decorator_list:
                    if is_mcp_tool_decorator(deco):
                        body = child.body
                        if body and isinstance(body[0], ast.Expr)\
                                and isinstance(body[0].value, ast.Constant)\
                                and isinstance(body[0].value.value, str):
                            doc = body[0].value.value
                            repl.append((deco, doc, ""))
            visit(child)

    visit(tree)
    return repl

def serialize_doc(doc: str) -> str:
    text = doc.strip()
    if "\n" in text:
        cleaned: list[str] = []
        for line in text.splitlines():
            stripped = line.strip()
            if stripped:
                cleaned.append(stripped)
        text = " ".join(cleaned)
    text = text.replace("\\", "\\\\").replace('"', '\\"')
    if len(text) > 700:
        text = text[:697] + "..."
    return f'"{text}"'

def patch_file(path: Path) -> int:
    src = path.read_text(encoding="utf-8")
    try:
        tree = ast.parse(src)
    except SyntaxError:
        return 0

    lines = src.splitlines(keepends=True)
    line_offsets = [0]
    acc = 0
    for ln in lines:
        acc += len(ln)
        line_offsets.append(acc)

    def to_offset(node: ast.AST) -> tuple[int, int]:
        return (
            line_offsets[node.lineno - 1] + node.col_offset,
            line_offsets[node.end_lineno - 1] + node.end_col_offset,
        )

    replacements: list[tuple[int, int, str]] = []
    docstring_removals: list[tuple[int, int]] = []

    def visit(node: ast.AST) -> None:
        for child in ast.iter_child_nodes(node):
            if isinstance(child, (ast.FunctionDef, ast.AsyncFunctionDef)):
                for deco in child.decorator_list:
                    if is_mcp_tool_decorator(deco):
                        body = child.body
                        if body and isinstance(body[0], ast.Expr)\
                                and isinstance(body[0].value, ast.Constant)\
                                and isinstance(body[0].value.value, str):
                            doc = body[0].value.value
                            new_call_args = serialize_doc(doc)
                            if isinstance(deco, ast.Call):
                                if deco.args or deco.keywords:
                                    has_desc = any(kw.arg == "description" for kw in deco.keywords)
                                    if has_desc:
                                        pass
                                    else:
                                        if deco.keywords:
                                            last = deco.keywords[-1]
                                            _, end = to_offset(last)
                                            replacements.append((
                                                end, end,
                                                f", description={new_call_args}",
                                            ))
                                        elif deco.args:
                                            last = deco.args[-1]
                                            _, end = to_offset(last)
                                            replacements.append((
                                                end, end,
                                                f", description={new_call_args}",
                                            ))
                                else:
                                    start, end = to_offset(deco)
                                    rng_start = line_offsets[deco.lineno - 1] + deco.col_offset
                                    paren_start = src.find("(", rng_start)
                                    paren_end = src.find(")", paren_start)
                                    if 0 <= paren_start < paren_end:
                                        replacements.append((
                                            paren_start + 1, paren_end,
                                            f"description={new_call_args}",
                                        ))
                            else:
                                start, end = to_offset(deco)
                                replacements.append((
                                    end, end,
                                    f"(description={new_call_args})",
                                ))
                            expr_start, expr_end = to_offset(body[0])
                            after = expr_end
                            while after < len(src) and src[after] in " \t":
                                after += 1
                            if after < len(src) and src[after] == "\n":
                                after += 1
                            docstring_removals.append((expr_start, after))
            visit(child)

    visit(tree)

    if not replacements and not docstring_removals:
        return 0

    edits: list[tuple[int, int, str, str]] = []
    for start, end, ins in replacements:
        edits.append((start, end, ins, "ins"))
    for start, end in docstring_removals:
        edits.append((start, end, "", "del"))
    edits.sort(key=lambda x: x[0], reverse=True)

    out = src
    for start, end, ins, kind in edits:
        if kind == "ins":
            out = out[:start] + ins + out[start:]
        else:
            line_start = out.rfind("\n", 0, start) + 1
            prefix = out[line_start:start]
            if prefix.strip() == "":
                start = line_start
            out = out[:start] + out[end:]

    path.write_text(out, encoding="utf-8")
    return len(replacements)

def main() -> int:
    if not TOOLS_DIR.exists():
        print(f"Tools dir not found: {TOOLS_DIR}")
        return 1
    total = 0
    for f in sorted(TOOLS_DIR.glob("*.py")):
        if f.name == "__init__.py":
            continue
        count = patch_file(f)
        if count:
            print(f"  migrated {count} tool docstrings in {f.name}")
            total += count
    print(f"\nDone. {total} MCP tool docstrings migrated to description= decorator args.")
    return 0

if __name__ == "__main__":
    sys.exit(main())
