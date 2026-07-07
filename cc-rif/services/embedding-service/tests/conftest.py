from __future__ import annotations

import sys
from pathlib import Path

# Ensure test imports can resolve local modules like app.py when pytest changes import mode.
ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))
