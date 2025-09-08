"""FLEXT - Enterprise Data Integration Platform.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT
"""


"""FLEXT FlexCore - Event-driven architecture system."""

from __future__ import annotations

from flexcore.core import FlexCore
from importlib.metadata import version

__version__ = version("flexcore")
__version_info__ = tuple(int(x) for x in __version__.split(".") if x.isdigit())
__author__ = "FLEXT Team"
__email__ = "team@flext.dev"


__all__: list[str] = ["FlexCore", "__version__", "__version_info__"]
