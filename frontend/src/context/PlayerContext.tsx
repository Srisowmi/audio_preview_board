import { createContext, useContext, type ReactNode } from "react";
import { useAudioPlayer, type PlayerApi } from "../hooks/useAudioPlayer";

const PlayerContext = createContext<PlayerApi | null>(null);

/** Provides the single shared audio player to the whole tree. */
export function PlayerProvider({ children }: { children: ReactNode }) {
  const player = useAudioPlayer();
  return <PlayerContext.Provider value={player}>{children}</PlayerContext.Provider>;
}

export function usePlayer(): PlayerApi {
  const ctx = useContext(PlayerContext);
  if (!ctx) throw new Error("usePlayer must be used inside <PlayerProvider>");
  return ctx;
}
