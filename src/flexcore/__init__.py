"""FLEXT FlexCore - Event-driven architecture system.

Copyright (c) 2025 Flext. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

from flexcore.core import FlexCore

__version__ = "0.9.0"  # TODO(team): Use importlib.metadata.version("flexcore") #123
__author__ = "FLEXT Team"
__email__ = "team@flext.dev"


__all__: list[str] = [
    "annotations", "FlexCore", "__version__",
] = ["FlexCore", "__version__"]
