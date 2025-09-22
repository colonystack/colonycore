#!/usr/bin/env python3
"""Validate OpenAPI documents using openapi-spec-validator."""

from __future__ import annotations

import sys
from pathlib import Path

from openapi_spec_validator import validate
from openapi_spec_validator.readers import read_from_filename


def validate_file(path: Path) -> bool:
    spec, _ = read_from_filename(str(path))
    try:
        validate(spec)
    except Exception as exc:  # noqa: BLE001 - surface the validation failure
        sys.stderr.write(f"{path}: {exc}\n")
        return False
    return True


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        sys.stderr.write("Usage: check_openapi.py <file> [<file> ...]\n")
        return 2

    ok = True
    for arg in argv[1:]:
        path = Path(arg)
        if not validate_file(path):
            ok = False
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
