"""Version and package metadata using importlib.metadata.

Single source of truth pattern following flext-core standards.
All metadata comes from pyproject.toml via importlib.metadata.

Copyright (c) 2025 client-a Telecom. Todos os direitos reservados.
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
__author__ = _metadata["Author"]
__author_email__ = _metadata["Author-Email"]
__license__ = _metadata["License"]
__url__ = _metadata["Home-Page"]

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
