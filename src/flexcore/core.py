"""FlexCore main class - Event-driven architecture core."""

from typing import Any


class FlexCore:
    """Main FlexCore class for event-driven architecture."""

    def __init__(self, config: dict[str, Any] | None = None) -> None:
        """Initialize FlexCore with optional configuration.

        Args:
            config: Optional configuration dictionary
        """
        self.config = config or {}
        self._initialized = False

    def initialize(self) -> None:
        """Initialize the FlexCore system."""
        if self._initialized:
            return

        # Initialize components here
        self._initialized = True

    def is_initialized(self) -> bool:
        """Check if FlexCore is initialized.

        Returns:
            True if initialized, False otherwise
        """
        return self._initialized

    def shutdown(self) -> None:
        """Shutdown the FlexCore system."""
        self._initialized = False
