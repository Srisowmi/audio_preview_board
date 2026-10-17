import { usePlayer } from "../context/PlayerContext";
import { FALLBACK_DURATION } from "../hooks/useAudioPlayer";
import { formatTime } from "../utils/formatTime";
import { ProgressBar } from "./ProgressBar";

/** Now-playing bar: play/pause, elapsed / total time and a seekable progress bar. */
export function PlayerBar() {
  const { track, isPlaying, isLoading, currentTime, duration, error, play, stop, seek } = usePlayer();

  if (!track) return null;

  const total = duration || FALLBACK_DURATION;

  return (
    <section className="player-bar" aria-label="Audio player">
      <img src={track.artworkUrl} alt="" width={48} height={48} />

      <div className="player-info">
        <span className="track-name">{track.name}</span>
        <span className="track-meta">{track.artist}</span>
      </div>

      <button type="button" onClick={() => play(track)} aria-label={`${isPlaying ? "Pause" : "Play"} ${track.name}`}>
        {isPlaying ? "Pause" : "Play"}
      </button>

      <span className="time" aria-hidden="true">
        {formatTime(currentTime)}
      </span>
      <ProgressBar value={currentTime} max={total} onSeek={seek} />
      <span className="time" aria-hidden="true">
        {formatTime(total)}
      </span>

      <button type="button" onClick={stop} aria-label="Close player">
        ✕
      </button>

      <div className="player-status" role={error ? "alert" : "status"}>
        {error ?? (isLoading ? "Loading preview…" : "")}
      </div>
    </section>
  );
}
