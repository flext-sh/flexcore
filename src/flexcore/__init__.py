"""FLEXT FlexCore - Event-driven architecture system.

Copyright (c) 2025 Flext. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

from flexcore.core import FlexCore
from importlib.metadata import version

__version__ = version("flexcore")
__author__ = "FLEXT Team"
__email__ = "team@flext.dev"


__all__: list[str] = ["FlexCore", "__version__"]
