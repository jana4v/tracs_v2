/**
 * Shared numeric / formatting / key utilities for tracs_nova panels.
 *
 * Centralized here to avoid duplication across calibration, spurious, and
 * channel panels that previously each re-implemented these helpers.
 */

/**
 * Coerce any value to a finite number. Returns 0 for non-finite / non-numeric inputs.
 * Use this when a numeric default is desired.
 */
export function toNumber(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

/**
 * Coerce any value to a finite number or null. Returns null for empty / invalid inputs.
 * Use this when an explicit "no value" must be distinguished from 0.
 */
export function toNumberOrNull(value: unknown): number | null {
  if (value === null || value === undefined) return null;
  const text = String(value).trim();
  if (text === '') return null;
  const n = Number(text);
  return Number.isFinite(n) ? n : null;
}

/**
 * Format a number for display: 3-decimal rounding, empty string for non-finite.
 */
export function formatNumber(value: number): string {
  if (!Number.isFinite(value)) return '';
  return String(Math.round(value * 1000) / 1000);
}

/**
 * AG-Grid valueFormatter that displays the absolute value of a numeric cell
 * (used for loss columns where the underlying value may be signed).
 */
export function absLossFormatter(params: { value: unknown }): string {
  const raw = params?.value;
  if (raw === '' || raw === null || raw === undefined) return '';
  const n = Number(raw);
  if (!Number.isFinite(n)) return String(raw);
  return formatNumber(Math.abs(n));
}

/**
 * Build a stable lookup key from `${code}|${port}|${frequencyHz}`.
 */
export function buildKey(code: string, port: string, frequencyHz: number | string): string {
  const f = toNumber(frequencyHz);
  return `${String(code).trim()}|${String(port).trim()}|${f}`;
}

/**
 * Free-space path loss in dB.
 *   FSPL = 20*log10(d) + 20*log10(f) + 20*log10(4π/c)
 * With d in meters and f in MHz (f_Hz = f_MHz * 1e6):
 *   20*log10(4π/c) + 20*log10(1e6) = -147.55 + 120 = -27.55
 * So:  FSPL = 20*log10(d_m) + 20*log10(f_MHz) - 27.55
 * Returns 0 if either input is non-positive.
 *
 * Mirrors backend `tracs_nova/src/utils/fspl.py::compute_fspl`.
 */
export function computeFspl(distanceMeters: number, frequencyMHz: number): number {
  if (!(distanceMeters > 0) || !(frequencyMHz > 0)) return 0;
  return 20 * Math.log10(distanceMeters) + 20 * Math.log10(frequencyMHz) - 27.55;
}
