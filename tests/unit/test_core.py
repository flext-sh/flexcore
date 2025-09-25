"""Comprehensive tests for FlexCore main class.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT
"""

from __future__ import annotations

from flexcore import FlexCore


class TestFlexCore:
    """Comprehensive test suite for FlexCore class."""

    def test_init_without_config(self) -> None:
        """Test FlexCore initialization without configuration."""
        core = FlexCore()

        assert core.config == {}
        assert core._initialized is False
        assert not core.is_initialized()

    def test_init_with_config(self) -> None:
        """Test FlexCore initialization with configuration."""
        config = {"setting1": "value1", "setting2": 42}
        core = FlexCore(config)

        assert core.config == config
        assert core._initialized is False
        assert not core.is_initialized()

    def test_init_with_none_config(self) -> None:
        """Test FlexCore initialization with None configuration."""
        core = FlexCore(None)

        assert core.config == {}
        assert core._initialized is False
        assert not core.is_initialized()

    def test_initialize_once(self) -> None:
        """Test FlexCore initialization process."""
        core = FlexCore()

        # Initially not initialized
        assert not core.is_initialized()

        # Initialize
        core.initialize()

        # Now initialized
        assert core.is_initialized()
        assert core._initialized is True

    def test_initialize_multiple_times(self) -> None:
        """Test that multiple initializations are handled correctly."""
        core = FlexCore()

        # Initialize multiple times
        core.initialize()
        assert core.is_initialized()

        core.initialize()  # Should not change state
        assert core.is_initialized()

        core.initialize()  # Should not change state
        assert core.is_initialized()

    def test_shutdown(self) -> None:
        """Test FlexCore shutdown process."""
        core = FlexCore()

        # Initialize first
        core.initialize()
        assert core.is_initialized()

        # Shutdown
        core.shutdown()
        assert not core.is_initialized()
        assert core._initialized is False

    def test_shutdown_without_init(self) -> None:
        """Test shutdown when not initialized."""
        core = FlexCore()

        # Shutdown without initialization
        core.shutdown()
        assert not core.is_initialized()
        assert core._initialized is False

    def test_lifecycle_workflow(self) -> None:
        """Test complete lifecycle workflow."""
        config = {"app_name": "test_app", "debug": True}
        core = FlexCore(config)

        # Initial state
        assert core.config == config
        assert not core.is_initialized()

        # Initialize
        core.initialize()
        assert core.is_initialized()

        # Shutdown
        core.shutdown()
        assert not core.is_initialized()

        # Re-initialize
        core.initialize()
        assert core.is_initialized()

    def test_config_immutability_after_init(self) -> None:
        """Test that config remains accessible after initialization."""
        config: dict[str, object] = {"database_url": "sqlite:///test.db"}
        core = FlexCore(config)

        core.initialize()

        # Config should still be accessible
        assert core.config == config
        assert core.config["database_url"] == "sqlite:///test.db"

    def test_complex_config_handling(self) -> None:
        """Test handling of complex configuration objects."""
        complex_config: dict[str, object] = {
            "database": {"host": "localhost", "port": 5432},
            "features": ["auth", "logging"],
            "limits": {"max_connections": 100, "timeout": 30},
        }

        core = FlexCore(complex_config)
        assert core.config == complex_config

        core.initialize()
        assert core.is_initialized()
        assert core.config == complex_config
