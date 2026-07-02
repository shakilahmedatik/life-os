# Changelog

All notable changes to LifeOS are documented here.

## [0.1.0.0] - 2026-07-03

### Added
- Command Center spine: live vertical timeline highlighting the current block, driven by a Go (Fiber) WebSocket ticker at 1s cadence.
- Sun–Thu 7-block schedule (workout, breakfast, Go/AI block, work AM, lunch, work PM, wind-down) plus Fri/Sat rest-day card.
- Pomodoro engine: 25-minute server-authoritative countdown (computed from `EndsAt`, no separate goroutine), start/stop via `/api/focus/start` and `/api/focus/stop`.
- Lazy-eval daily reset: the day materializes on first `GET /api/today`, no cron.
- Planned-vs-actual block model: every `Block` carries `Planned` and `Actual` shapes (manual overrun entry now, observed-state telemetry later).
- Free-time card with countdown to next block (server Dhaka-time, correct regardless of client timezone).
- WebSocket hub with 500ms write deadlines; reconnect-on-close client.
- Table-driven Go tests for `buildDay`, `currentBlock` (10 boundary cases), Pomodoro remaining, lazy-eval cache.
- `make dev` launcher and Vite proxy with `ws: true`.

### Fixed
- Data race in `handleToday` (marshal under RLock snapshot).
- Data race in tick-loop Pomodoro read (via `Store.pomoRemain` under RLock).
- `handleFocusStop` now clears stale pomodoros on ended blocks.
- `prevBlock` reset on date change (latent cross-day bug).
- `time.LoadLocation` fallback to `FixedZone("BDT")` for stripped-down containers.
- Time formatting: durations ≥1h now render `Hh Mm`; rest-day card shows "all day" instead of a 22-hour countdown.
- Pomodoro UI: clearly labeled `MM:SS / 25:00` with a stop button; "start focus (25 min)" button hidden on rest-day block.
