import type { ReactNode } from "react";

interface Props {
  kind: "idle" | "loading" | "empty" | "error";
  children: ReactNode;
  onRetry?: () => void;
}

/** Explicit idle / loading / empty / error states. Errors are announced assertively. */
export function StatusMessage({ kind, children, onRetry }: Props) {
  return (
    <div className={`status status-${kind}`} role={kind === "error" ? "alert" : "status"}>
      <p>{children}</p>
      {kind === "error" && onRetry && (
        <button type="button" onClick={onRetry}>
          Try again
        </button>
      )}
    </div>
  );
}
