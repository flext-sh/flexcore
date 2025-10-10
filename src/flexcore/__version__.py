"""Version and package metadata using importlib.metadata.

Single source of truth pattern following flext-core standards.
All metadata comes from pyproject.toml via importlib.metadata.

Copyright (c) 2025 Algar Telecom. Todos os direitos reservados.
SPDX-License-Identifier: Proprietary
"""

from __future__ import annotations

from importlib.metadata import metadata

_metadata = metadata("flexcore")

__version__ = _metadata["Version"]
__version_info__ = tuple(
    int(part) if part.isdigit() else part for part in __version__.split(".")
)
__title__ = _metadata["Name"]
__description__ = _metadata["Summary"]
__author__ = _metadata.get("Author")
__author_email__ = _metadata.get("Author-Email")
__license__ = _metadata.get("License")
__url__ = _metadata.get("Home-Page")

__all__ = [
    "__author__",
    "__author_email__",
    "__description__",
    "__license__",
    "__title__",
    "__url__",
    "__version__",
    "__version_info__",
]
