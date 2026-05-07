"""Free-space path loss helper.

Single source of truth for FSPL computation across the backend. The GUI
mirrors the same formula in
``GUI/app/components/TracsV2/Database/Calibration/TransmitterCalibrationPanel.vue``
(``computeFspl``); keep them in sync.
"""

from __future__ import annotations

import math


def compute_fspl(distance_meters: float, frequency_mhz: float) -> float:
    """Return free-space path loss in dB.

    ``FSPL = 20*log10(d_m) + 20*log10(f_MHz) - 27.55``

    Returns 0.0 if either input is non-positive (matches GUI behavior).
    """
    if not (distance_meters > 0) or not (frequency_mhz > 0):
        return 0.0
    return 20.0 * math.log10(distance_meters) + 20.0 * math.log10(frequency_mhz) - 27.55
