#!/usr/bin/env python3
# Owner-Skill: .claude/skills/scripts-maintenance/SKILL.md
"""Documentation validation script for FLEXT projects.

Checks ADR skill references and generates validation reports per project scope.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from documentation.shared import (
    Scope,
    build_scopes,
    write_json,
    write_markdown,
)


def has_adr_reference(skill_path: Path) -> bool:
    """Check whether a skill file contains an ADR reference."""
    text = skill_path.read_text(encoding="utf-8", errors="ignore").lower()
    return "adr" in text


def run_adr_skill_check(root: Path) -> tuple[int, list[str]]:
    """Run ADR skill check and return exit code with missing skill names."""
    skills_root = root / ".claude/skills"
    required: list[str] = []
    config = root / "docs/architecture/architecture_config.json"
    if config.exists():
        payload = json.loads(config.read_text(encoding="utf-8", errors="ignore"))
        docs_validation = payload.get("docs_validation", {})
        configured = docs_validation.get("required_skills", [])
        if isinstance(configured, list):
            required = [item for item in configured if isinstance(item, str) and item]
    if not required:
        required = ["rules-docs", "scripts-maintenance", "readme-standardization"]

    missing: list[str] = []
    for name in required:
        skill = skills_root / name / "SKILL.md"
        if not skill.exists() or not has_adr_reference(skill):
            missing.append(name)
    return (0 if not missing else 1), missing


def maybe_write_todo(scope: Scope, *, apply_mode: bool) -> bool:
    """Write a TODOS.md file for the scope if apply mode is enabled."""
    if scope.name == "root" or not apply_mode:
        return False
    path = scope.path / "TODOS.md"
    content = "# TODOS\n\n- [ ] Resolve documentation validation findings from `.reports/docs/validate-report.md`.\n"
    path.write_text(content, encoding="utf-8")
    return True


def run_scope(scope: Scope, *, apply_mode: bool, check: str) -> int:
    """Run validation for a single project scope and write reports."""
    status = "OK"
    message = "validation passed"
    details: dict[str, object] = {}
    config_exists = (scope.path / "docs/architecture/architecture_config.json").exists()
    if scope.name == "root" and config_exists and check in {"adr-skill", "all"}:
        code, missing = run_adr_skill_check(scope.path)
        details["missing_adr_skills"] = missing
        if code != 0:
            status = "FAIL"
            message = f"missing adr references in skills: {', '.join(missing)}"
    wrote_todo = maybe_write_todo(scope, apply_mode=apply_mode)
    details["todo_written"] = wrote_todo
    write_json(
        scope.report_dir / "validate-summary.json",
        {
            "summary": {
                "scope": scope.name,
                "result": status,
                "message": message,
                "apply": apply_mode,
            },
            "details": details,
        },
    )
    write_markdown(
        scope.report_dir / "validate-report.md",
        [
            "# Docs Validate Report",
            "",
            f"Scope: {scope.name}",
            f"Result: {status}",
            f"Message: {message}",
            f"TODO written: {int(wrote_todo)}",
        ],
    )
    print(f"PROJECT={scope.name} PHASE=validate RESULT={status} REASON={message}")
    return 1 if status == "FAIL" else 0


def main() -> int:
    """Entry point for documentation validation CLI."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=".")
    parser.add_argument("--project")
    parser.add_argument("--projects")
    parser.add_argument("--output-dir", default=".reports/docs")
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--check", default="all")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    scopes = build_scopes(
        root=root,
        project=args.project,
        projects=args.projects,
        output_dir=args.output_dir,
    )
    failures = 0
    for scope in scopes:
        failures += run_scope(scope, apply_mode=args.apply, check=args.check)
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
