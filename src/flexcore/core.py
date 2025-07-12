"""FlexCore main class - Event-driven architecture core."""

from typing import Any


class FlexCore:
    """Main FlexCore class for event-driven architecture."""

    def __init__(self, config: dict[str, Any] | None = None) -> None:
        self.config = config or {}
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
