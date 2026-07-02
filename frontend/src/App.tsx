import { useEffect, useRef, useState } from 'react'

type Block = {
  id: string
  label: string
  start: string
  end: string
  planned: { duration_min: number }
  actual: {
    start?: string
    end?: string
    overrun_min: number
    focus_quality?: number
  }
  has_pomodoro: boolean
}

type Day = {
  date: string
  blocks: Block[]
}

type TickMsg = {
  type: string
  date: string
  block_id: string | null
  seconds_remaining_in_block: number | null
  pomodoro_seconds_remaining: number | null
  next_block_start: string | null
}

const POMODORO_TOTAL = 25 * 60

function fmtDuration(s: number): string {
  if (s < 3600) {
    const m = Math.floor(s / 60)
    const sec = s % 60
    return `${m}:${sec.toString().padStart(2, '0')}`
  }
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  return `${h}h ${m}m`
}

function App() {
  const [day, setDay] = useState<Day | null>(null)
  const [tick, setTick] = useState<TickMsg | null>(null)
  const [pomoRunning, setPomoRunning] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const currentDateRef = useRef<string>('')

  // Fetch today on mount
  async function fetchDay(date?: string) {
    const res = await fetch('/api/today')
    const d: Day = await res.json()
    if (date && date !== currentDateRef.current) {
      currentDateRef.current = date
      setDay(d)
    } else if (!date) {
      currentDateRef.current = d.date
      setDay(d)
    }
  }

  useEffect(() => {
    fetchDay()
    connectWS()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function connectWS() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/ws`)
    wsRef.current = ws

    ws.onopen = () => {
      fetchDay()
    }

    ws.onmessage = (ev) => {
      try {
        const msg: TickMsg = JSON.parse(ev.data)
        if (msg.type === 'tick') {
          if (msg.date !== currentDateRef.current) {
            fetchDay(msg.date)
          }
          setTick(msg)
          if (msg.pomodoro_seconds_remaining === null || msg.pomodoro_seconds_remaining === 0) {
            setPomoRunning(false)
          } else {
            setPomoRunning(true)
          }
        }
      } catch {
        // ignore malformed
      }
    }

    ws.onclose = () => {
      setTimeout(() => connectWS(), 1000)
    }

    ws.onerror = () => {
      ws.close()
    }
  }

  async function startFocus(blockId: string) {
    try {
      const res = await fetch('/api/focus/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ block_id: blockId }),
      })
      if (res.ok) setPomoRunning(true)
    } catch {
      // ignore
    }
  }

  async function stopFocus() {
    try {
      await fetch('/api/focus/stop', { method: 'POST' })
      setPomoRunning(false)
    } catch {
      // ignore
    }
  }

  async function logOverrun(blockId: string) {
    const input = prompt('Overrun in minutes:', '5')
    if (input === null) return
    const n = parseInt(input, 10)
    if (isNaN(n) || n < 0) return
    try {
      await fetch('/api/actual', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ block_id: blockId, overrun_min: n }),
      })
      fetchDay()
    } catch {
      // ignore
    }
  }

  const currentBlockId = tick?.block_id ?? null
  const pomoRem = tick?.pomodoro_seconds_remaining ?? null
  const blockRem = tick?.seconds_remaining_in_block ?? null

  if (!day) {
    return (
      <div className="flex h-full items-center justify-center text-neutral-500">
        loading...
      </div>
    )
  }

  // Free-time card: no current block. next_block_start comes from the server
  // (Dhaka time) so it's correct regardless of the client's timezone.
  const noCurrent = currentBlockId === null
  const nextStart = tick?.next_block_start ?? null

  return (
    <div className="min-h-full max-w-2xl mx-auto px-6 py-8">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-100">LifeOS</h1>
          <p className="text-xs text-neutral-500 mt-0.5">{day.date}</p>
        </div>
        {pomoRunning && pomoRem !== null && (
          <div className="text-right">
            <div className="flex items-center gap-2 justify-end">
              <span className="text-xl font-mono tabular-nums text-emerald-400">
                {fmtDuration(pomoRem)}
              </span>
              <button
                onClick={stopFocus}
                className="rounded bg-neutral-800 px-1.5 py-0.5 text-[10px] text-neutral-400 hover:bg-neutral-700 hover:text-neutral-200"
                title="Stop pomodoro"
              >
                stop
              </button>
            </div>
            <div className="text-[10px] text-neutral-600 uppercase tracking-wider">
              pomodoro · {fmtDuration(pomoRem)} / {fmtDuration(POMODORO_TOTAL)}
            </div>
          </div>
        )}
      </header>

      {noCurrent && (
        <div className="mb-6 rounded-lg border border-neutral-800 bg-neutral-900/50 p-5 text-center">
          <div className="text-sm text-neutral-400">Free time</div>
          {nextStart ? (
            <div className="mt-1 text-xs text-neutral-500">
              next block at <span className="text-neutral-300">{nextStart}</span>
              {blockRem === null && tick && (
                <span className="block mt-1 text-neutral-600">live</span>
              )}
            </div>
          ) : (
            <div className="mt-1 text-xs text-neutral-500">no more blocks today</div>
          )}
        </div>
      )}

      <div className="space-y-2">
        {day.blocks.map((b) => {
          const isCurrent = b.id === currentBlockId
          const isPast = b.actual.end !== undefined

          return (
            <div
              key={b.id}
              ref={isCurrent ? (el) => el?.scrollIntoView({ behavior: 'smooth', block: 'center' }) : undefined}
              className={[
                'rounded-lg border p-4 transition-all',
                isCurrent
                  ? 'border-emerald-500/60 bg-emerald-950/20 shadow-[0_0_12px_-2px_rgba(16,185,129,0.3)]'
                  : isPast
                    ? 'border-neutral-900 bg-neutral-950 opacity-40'
                    : 'border-neutral-800 bg-neutral-950/50',
              ].join(' ')}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {isCurrent && (
                    <span className="relative flex h-2.5 w-2.5">
                      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                      <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500" />
                    </span>
                  )}
                  <span className={isCurrent ? 'text-neutral-100 font-medium' : 'text-neutral-400'}>
                    {b.label}
                  </span>
                </div>
                <div className="text-xs font-mono tabular-nums text-neutral-500">
                  {b.start}–{b.end}
                </div>
              </div>

              {isCurrent && (
                <div className="mt-2 flex items-center justify-between">
                  <div className="text-xs text-neutral-500">
                    {b.id === 'rest'
                      ? 'all day'
                      : blockRem !== null
                        ? `${fmtDuration(blockRem)} left in block`
                        : ''}
                  </div>
                  {b.id !== 'rest' && (
                    <button
                      onClick={() => startFocus(b.id)}
                      disabled={pomoRunning}
                      className={[
                        'rounded px-3 py-1 text-xs font-medium transition-colors',
                        pomoRunning
                          ? 'bg-neutral-800 text-neutral-600 cursor-not-allowed'
                          : 'bg-emerald-600 text-white hover:bg-emerald-500',
                      ].join(' ')}
                    >
                      {pomoRunning ? 'focusing' : 'start focus (25 min)'}
                    </button>
                  )}
                </div>
              )}

              {isPast && b.actual.overrun_min > 0 && (
                <div className="mt-1.5">
                  <span className="inline-block rounded bg-red-950/60 px-1.5 py-0.5 text-[10px] font-medium text-red-400">
                    +{b.actual.overrun_min} min
                  </span>
                </div>
              )}

              {isPast && (
                <button
                  onClick={() => logOverrun(b.id)}
                  className="mt-1.5 text-[10px] text-neutral-600 hover:text-neutral-400"
                >
                  log overrun
                </button>
              )}
            </div>
          )
        })}
      </div>

      <footer className="mt-8 text-center text-[10px] text-neutral-700">
        LifeOS spine · Go (Fiber) + React · WebSocket live
      </footer>
    </div>
  )
}

export default App
