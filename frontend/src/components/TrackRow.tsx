import { usePlayer } from "../context/PlayerContext";
import type { Track } from "../types";
import { formatTime } from "../utils/formatTime";

interface ResultProps {
  variant: "result";
  track: Track;
  inShortlist: boolean;
  onAdd: (track: Track) => void;
}

interface ShortlistProps {
  variant: "shortlist";
  track: Track;
  isFirst: boolean;
  isLast: boolean;
  onRemove: (track: Track) => void;
  onMove: (track: Track, direction: "up" | "down") => void;
}

export function TrackRow(props: ResultProps | ShortlistProps) {
  const { track } = props;
  const player = usePlayer();
  const isCurrent = player.track?.id === track.id;
  const isPlaying = isCurrent && player.isPlaying;

  return (
    <li className={`track-row${isCurrent ? " is-current" : ""}`}>
      <img src={track.artworkUrl} alt="" width={56} height={56} loading="lazy" />

      <div className="track-info">
        <span className="track-name">{track.name}</span>
        <span className="track-meta">
          {track.artist}
          {track.album ? ` · ${track.album}` : ""}
        </span>
      </div>

      <span className="track-duration" aria-label={`Duration ${formatTime(track.durationMs / 1000)}`}>
        {formatTime(track.durationMs / 1000)}
      </span>

      <div className="track-actions">
        <button
          type="button"
          onClick={() => player.play(track)}
          aria-label={`${isPlaying ? "Pause" : "Play"} ${track.name} by ${track.artist}`}
        >
          {isPlaying ? "Pause" : "Play"}
        </button>

        {props.variant === "result" ? (
          props.inShortlist ? (
            <button type="button" disabled aria-label={`${track.name} is already in your shortlist`}>
              Added
            </button>
          ) : (
            <button type="button" onClick={() => props.onAdd(track)} aria-label={`Add ${track.name} to shortlist`}>
              Add
            </button>
          )
        ) : (
          <>
            <button
              type="button"
              data-track-id={track.id}
              data-action="up"
              disabled={props.isFirst}
              onClick={() => props.onMove(track, "up")}
              aria-label={`Move ${track.name} up`}
            >
              ↑
            </button>
            <button
              type="button"
              data-track-id={track.id}
              data-action="down"
              disabled={props.isLast}
              onClick={() => props.onMove(track, "down")}
              aria-label={`Move ${track.name} down`}
            >
              ↓
            </button>
            <button type="button" onClick={() => props.onRemove(track)} aria-label={`Remove ${track.name} from shortlist`}>
              Remove
            </button>
          </>
        )}
      </div>
    </li>
  );
}
