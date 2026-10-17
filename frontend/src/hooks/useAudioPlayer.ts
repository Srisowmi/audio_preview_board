import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Track } from "../types";

/** iTunes previews are 30s clips; used until the audio metadata reports the real length. */
export const FALLBACK_DURATION = 30;

export interface PlayerApi {
  track: Track | null;
  isPlaying: boolean;
  isLoading: boolean;
  currentTime: number;
  duration: number;
  error: string | null;
  /** Plays the track; if it is already the current track, toggles play/pause. */
  play: (track: Track) => void;
  /** Stops playback and clears the current track. */
  stop: () => void;
  seek: (seconds: number) => void;
}

/**
 * Owns the one and only HTMLAudioElement, which is what guarantees that a
 * single track plays at a time: starting a new track replaces the source.
 */
export function useAudioPlayer(): PlayerApi {
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const currentIdRef = useRef<number | null>(null);

  const [track, setTrack] = useState<Track | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const audio = new Audio();
    audio.preload = "auto";
    audioRef.current = audio;

    const readDuration = () => {
      if (Number.isFinite(audio.duration) && audio.duration > 0) setDuration(audio.duration);
    };
    const onTimeUpdate = () => setCurrentTime(audio.currentTime);
    const onPlay = () => setIsPlaying(true);
    const onPause = () => setIsPlaying(false);
    const onPlaying = () => setIsLoading(false);
    const onWaiting = () => setIsLoading(true);
    const onEnded = () => {
      setIsPlaying(false);
      setCurrentTime(audio.duration || 0);
    };
    const onError = () => {
      if (!audio.getAttribute("src")) return; // fired because we cleared the source
      setIsPlaying(false);
      setIsLoading(false);
      setError("This preview could not be played.");
    };

    audio.addEventListener("loadedmetadata", readDuration);
    audio.addEventListener("durationchange", readDuration);
    audio.addEventListener("timeupdate", onTimeUpdate);
    audio.addEventListener("play", onPlay);
    audio.addEventListener("pause", onPause);
    audio.addEventListener("playing", onPlaying);
    audio.addEventListener("waiting", onWaiting);
    audio.addEventListener("ended", onEnded);
    audio.addEventListener("error", onError);

    return () => {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
      audio.removeEventListener("loadedmetadata", readDuration);
      audio.removeEventListener("durationchange", readDuration);
      audio.removeEventListener("timeupdate", onTimeUpdate);
      audio.removeEventListener("play", onPlay);
      audio.removeEventListener("pause", onPause);
      audio.removeEventListener("playing", onPlaying);
      audio.removeEventListener("waiting", onWaiting);
      audio.removeEventListener("ended", onEnded);
      audio.removeEventListener("error", onError);
      audioRef.current = null;
    };
  }, []);

  const startPlayback = useCallback((audio: HTMLAudioElement) => {
    audio.play().catch((err: unknown) => {
      // AbortError = interrupted by a newer load/pause, which is expected.
      if (err instanceof DOMException && err.name === "AbortError") return;
      setIsPlaying(false);
      setIsLoading(false);
      setError("Playback was blocked or failed. Press play to try again.");
    });
  }, []);

  const play = useCallback(
    (next: Track) => {
      const audio = audioRef.current;
      if (!audio) return;

      if (currentIdRef.current === next.id) {
        if (audio.paused) {
          if (audio.ended) audio.currentTime = 0;
          setError(null);
          startPlayback(audio);
        } else {
          audio.pause();
        }
        return;
      }

      audio.pause();
      currentIdRef.current = next.id;
      setTrack(next);
      setCurrentTime(0);
      setDuration(0);
      setError(null);
      setIsLoading(true);
      audio.src = next.previewUrl;
      startPlayback(audio);
    },
    [startPlayback],
  );

  const stop = useCallback(() => {
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    }
    currentIdRef.current = null;
    setTrack(null);
    setIsPlaying(false);
    setIsLoading(false);
    setCurrentTime(0);
    setDuration(0);
    setError(null);
  }, []);

  const seek = useCallback((seconds: number) => {
    const audio = audioRef.current;
    if (!audio || !currentIdRef.current) return;
    const max = Number.isFinite(audio.duration) && audio.duration > 0 ? audio.duration : FALLBACK_DURATION;
    const clamped = Math.min(Math.max(seconds, 0), max);
    audio.currentTime = clamped;
    setCurrentTime(clamped);
  }, []);

  return useMemo(
    () => ({ track, isPlaying, isLoading, currentTime, duration, error, play, stop, seek }),
    [track, isPlaying, isLoading, currentTime, duration, error, play, stop, seek],
  );
}
