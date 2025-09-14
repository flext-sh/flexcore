"""FlexCore pytest configuration.

Configures pytest to work properly with the FlexCore Go project structure.

Copyright (c) 2025 Flext. All rights reserved.
SPDX-License-Identifier: MIT
"""

from __future__ import annotations

import sys
from pathlib import Path

# Add flexcore/src to Python path for imports
flexcore_src = Path(__file__).parent / "src"
if str(flexcore_src) not in sys.path:
    sys.path.insert(0, str(flexcore_src))
