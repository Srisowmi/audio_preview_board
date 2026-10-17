import { useState, type KeyboardEvent, type PointerEvent } from "react";
import { formatTime } from "../utils/formatTime";

interface Props {
  value: number;
  max: number;
  onSeek: (seconds: number) => void;
}

const STEP = 5;
const BIG_STEP = 10;

/** Seekable progress bar: click, drag, or use the keyboard (a WAI-ARIA slider). */
export function ProgressBar({ value, max, onSeek }: Props) {
  const [dragValue, setDragValue] = useState<number | null>(null);
  const enabled = max > 0;
  const shown = Math.min(dragValue ?? value, max);
  const percent = enabled ? (shown / max) * 100 : 0;

  const valueFromPointer = (e: PointerEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    if (rect.width === 0) return 0;
    const ratio = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
    return ratio * max;
  };

  const onPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    if (!enabled) return;
    e.currentTarget.setPointerCapture(e.pointerId);
    e.currentTarget.focus();
    setDragValue(valueFromPointer(e));
  };

  const onPointerMove = (e: PointerEvent<HTMLDivElement>) => {
    if (e.currentTarget.hasPointerCapture(e.pointerId)) setDragValue(valueFromPointer(e));
  };

  const onPointerUp = (e: PointerEvent<HTMLDivElement>) => {
    if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
    e.currentTarget.releasePointerCapture(e.pointerId);
    setDragValue(null);
    onSeek(valueFromPointer(e)); // audio seeks once, on release
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (!enabled) return;
    let next: number | null = null;
    switch (e.key) {
      case "ArrowLeft":
      case "ArrowDown":
        next = value - STEP;
        break;
      case "ArrowRight":
      case "ArrowUp":
        next = value + STEP;
        break;
      case "PageDown":
        next = value - BIG_STEP;
        break;
      case "PageUp":
        next = value + BIG_STEP;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = max;
        break;
    }
    if (next !== null) {
      e.preventDefault();
      onSeek(Math.min(Math.max(next, 0), max));
    }
  };

  return (
    <div
      className="progress"
      role="slider"
      tabIndex={0}
      aria-label="Seek"
      aria-valuemin={0}
      aria-valuemax={Math.round(max)}
      aria-valuenow={Math.round(shown)}
      aria-valuetext={`${formatTime(shown)} of ${formatTime(max)}`}
      aria-disabled={!enabled}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={() => setDragValue(null)}
      onKeyDown={onKeyDown}
    >
      <div className="progress-track">
        <div className="progress-fill" style={{ width: `${percent}%` }} />
        <div className="progress-thumb" style={{ left: `${percent}%` }} />
      </div>
    </div>
  );
}
