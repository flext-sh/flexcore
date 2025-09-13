"""FLEXT FlexCore - Event-driven architecture system.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT
"""

from __future__ import annotations

from importlib.metadata import version

from flexcore.core import FlexCore

__version__ = version("flexcore")
__version_info__ = tuple(int(x) for x in __version__.split(".") if x.isdigit())
__author__ = "FLEXT Team"
__email__ = "team@flext.dev"


__all__: list[str] = ["FlexCore", "__version__", "__version_info__"]
