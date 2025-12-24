"""FLEXT FlexCore - Event-driven architecture system.

Copyright (c) 2025 FLEXT Team. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

from flext import FlextConstants

from flexcore.__version__ import __version__, __version_info__
from flexcore.core import FlexCore

__name__ = "flexcore"
__author__ = "FLEXT Team"
__email__ = "team@flext.dev"

__all__ = [
    "FlexCore",
    "FlextConstants",
    "__author__",
    "__email__",
    "__name__",
    "__version__",
    "__version_info__",
]
