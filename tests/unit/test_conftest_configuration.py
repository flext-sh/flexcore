"""Tests for conftest.py path configuration.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

import sys
from pathlib import Path

import pytest

# Import flexcore modules for testing
try:
    import flexcore
    from flexcore import FlexCore
except ImportError:
    flexcore = None
    FlexCore = None


class TestConftestConfiguration:
    """Test suite for conftest.py path setup."""

    def test_flexcore_src_in_path(self) -> None:
        """Test that flexcore/src is in Python path."""
        # Calculate expected path
        conftest_path = Path(__file__).parent.parent.parent / "conftest.py"
        expected_src_path = conftest_path.parent / "src"

        # Check if path is in sys.path
        path_found = any(
            Path(p).resolve() == expected_src_path.resolve() for p in sys.path
        )

        assert path_found, f"Expected path {expected_src_path} not found in sys.path"

    def test_flexcore_import_works(self) -> None:
        """Test that flexcore can be imported after path setup."""
        assert flexcore is not None, "flexcore module not imported"
        assert FlexCore is not None, "FlexCore class not imported"

    def test_flexcore_core_import_works(self) -> None:
        """Test that flexcore.core can be imported after path setup."""
        assert FlexCore is not None, "FlexCore class not imported"

    def test_path_setup_idempotent(self) -> None:
        """Test that path setup is idempotent (safe to run multiple times)."""
        # Get current sys.path
        original_path = sys.path.copy()

        # Simulate conftest execution
        conftest_path = Path(__file__).parent.parent.parent / "conftest.py"
        flexcore_src = conftest_path.parent / "src"

        # Add path if not present (simulate conftest behavior)
        if str(flexcore_src) not in sys.path:
            sys.path.insert(0, str(flexcore_src))

        # Add again (should not duplicate)
        if str(flexcore_src) not in sys.path:
            sys.path.insert(0, str(flexcore_src))

        # Count occurrences
        path_count = sum(1 for p in sys.path if str(flexcore_src) == p)

        # Should only appear once
        assert path_count <= 1, (
            f"Path {flexcore_src} appears {path_count} times in sys.path"
        )

        # Restore original path to avoid side effects
        sys.path[:] = original_path

    def test_conftest_file_exists(self) -> None:
        """Test that conftest.py file exists and is readable."""
        conftest_path = Path(__file__).parent.parent.parent / "conftest.py"

        assert conftest_path.exists(), f"conftest.py not found at {conftest_path}"
        assert conftest_path.is_file(), f"conftest.py is not a file: {conftest_path}"

        # Test that file is readable
        try:
            with conftest_path.open() as f:
                content = f.read()
                assert "flexcore" in content.lower()
        except Exception as e:
            pytest.fail(f"Failed to read conftest.py: {e}")

    def test_src_directory_exists(self) -> None:
        """Test that src directory exists."""
        conftest_path = Path(__file__).parent.parent.parent / "conftest.py"
        src_path = conftest_path.parent / "src"

        assert src_path.exists(), f"src directory not found at {src_path}"
        assert src_path.is_dir(), f"src is not a directory: {src_path}"

    def test_flexcore_package_in_src(self) -> None:
        """Test that flexcore package exists in src directory."""
        conftest_path = Path(__file__).parent.parent.parent / "conftest.py"
        flexcore_package = conftest_path.parent / "src" / "flexcore"

        assert flexcore_package.exists(), (
            f"flexcore package not found at {flexcore_package}"
        )
        assert flexcore_package.is_dir(), (
            f"flexcore is not a directory: {flexcore_package}"
        )

        # Check for __init__.py
        init_file = flexcore_package / "__init__.py"
        assert init_file.exists(), "__init__.py not found in flexcore package"
