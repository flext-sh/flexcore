"""FLEXT FlexCore - Event-driven architecture system.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

from flext_core import FlextConstants

from flexcore.__version__ import __version__, __version_info__
from flexcore.core import FlexCore

__all__ = [
    "FlexCore",
    "FlextConstants",
    "__version__",
    "__version_info__",
]
