"""Comprehensive tests for FlexCore package initialization.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT
"""

from __future__ import annotations

import re

import flexcore
from flexcore import FlexCore


class TestFlexCorePackage:
    """Test suite for FlexCore package initialization."""

    def test_package_exports(self) -> None:
        """Test that package exports are available."""
        assert hasattr(flexcore, "FlexCore")
        assert hasattr(flexcore, "__version__")
        assert hasattr(flexcore, "__version_info__")

    def test_version_attributes(self) -> None:
        """Test version attributes are properly set."""
        assert isinstance(flexcore.__version__, str)
        assert isinstance(flexcore.__version_info__, tuple)

        # Version should be in semantic versioning format
        version_pattern = re.compile(r"^\d+\.\d+\.\d+")
        assert version_pattern.match(flexcore.__version__)

    def test_version_info_structure(self) -> None:
        """Test version_info tuple structure."""
        version_info = flexcore.__version_info__
        assert isinstance(version_info, tuple)
        assert len(version_info) >= 3

        # All elements should be integers
        for part in version_info:
            assert isinstance(part, int)

    def test_author_and_email(self) -> None:
        """Test author and email attributes."""
        assert hasattr(flexcore, "__author__")
        assert hasattr(flexcore, "__email__")
        assert flexcore.__author__ == "FLEXT Team"
        assert flexcore.__email__ == "team@flext.dev"

    def test_all_attribute(self) -> None:
        """Test __all__ attribute is properly defined."""
        assert hasattr(flexcore, "__all__")
        assert isinstance(flexcore.__all__, list)

        expected_exports = ["FlexCore", "__version__", "__version_info__"]
        assert flexcore.__all__ == expected_exports

    def test_flexcore_class_accessible(self) -> None:
        """Test that FlexCore class is accessible through package import."""
        # Direct import should work
        core = flexcore.FlexCore()
        assert isinstance(core, FlexCore)

    def test_version_consistency(self) -> None:
        """Test version consistency between string and tuple."""
        version_str = flexcore.__version__
        version_tuple = flexcore.__version_info__

        # Extract numbers from version string
        version_parts = [int(x) for x in version_str.split(".") if x.isdigit()]

        # Should match version_info
        assert tuple(version_parts) == version_tuple

    def test_package_metadata(self) -> None:
        """Test package metadata is accessible."""
        # Test that package has standard attributes
        assert hasattr(flexcore, "__name__")
        assert flexcore.__name__ == "flexcore"

    def test_import_from_package(self) -> None:
        """Test importing specific items from package."""
        from flexcore import FlexCore, __version__, __version_info__  # noqa: PLC0415

        assert FlexCore is not None
        assert __version__ is not None
        assert __version_info__ is not None

    def test_core_module_import(self) -> None:
        """Test that core module is properly imported."""
        from flexcore.core import FlexCore as CoreFlexCore  # noqa: PLC0415

        # Should be the same class
        assert flexcore.FlexCore is CoreFlexCore
