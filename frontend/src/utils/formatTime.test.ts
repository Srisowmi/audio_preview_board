import { describe, expect, it } from "vitest";
import { formatTime } from "./formatTime";

describe("formatTime", () => {
  it("formats seconds as m:ss", () => {
    expect(formatTime(0)).toBe("0:00");
    expect(formatTime(5)).toBe("0:05");
    expect(formatTime(30)).toBe("0:30");
    expect(formatTime(65)).toBe("1:05");
    expect(formatTime(269)).toBe("4:29");
  });

  it("floors fractional seconds", () => {
    expect(formatTime(29.99)).toBe("0:29");
  });

  it("treats invalid input as 0:00", () => {
    expect(formatTime(NaN)).toBe("0:00");
    expect(formatTime(Infinity)).toBe("0:00");
    expect(formatTime(-3)).toBe("0:00");
  });
});
