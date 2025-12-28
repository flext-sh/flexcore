"""FlexCore main class - Event-driven architecture core.

Copyright (c) 2025 Flext. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

from typing import override

from flext_core.typings import t


class FlexCore:
    """Main FlexCore class for event-driven architecture."""

    @override
    def __init__(
        self,
        config: dict[str, t.Types.GeneralValueType] | None = None,
        **kwargs: object,
    ) -> None:
        """Initialize FlexCore with optional configuration."""
        self.config: dict[str, t.Types.GeneralValueType] = config or {}
        self._initialized = False

    def initialize(self) -> None:
        """Initialize the core system components."""
        if self._initialized:
            return

        # Initialize components here
        self._initialized = True

    def is_initialized(self) -> bool:
        """Check if system is initialized."""
        return self._initialized

    def shutdown(self) -> None:
        """Shutdown the core system."""
        self._initialized = False
