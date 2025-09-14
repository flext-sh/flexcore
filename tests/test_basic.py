"""Basic test template.

Copyright (c) 2025 Flext. All rights reserved.
SPDX-License-Identifier: MIT

"""

from __future__ import annotations

import sys
from pathlib import Path

# Add flexcore/src to Python path for imports
flexcore_src = Path(__file__).parent.parent / "src"
if str(flexcore_src) not in sys.path:
    sys.path.insert(0, str(flexcore_src))

from flexcore import FlexCore


def test_basic() -> None:
    """Test basic functionality."""
    assert True


def test_flexcore_import() -> None:
    """Test that FlexCore can be imported."""
    assert FlexCore is not None
