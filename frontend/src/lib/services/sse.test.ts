import { describe, expect, it } from "vitest";
import { isSessionEventsURL } from "./sse";

describe("isSessionEventsURL", () => {
  it("matches session SSE endpoints", () => {
    expect(
      isSessionEventsURL("http://localhost:8080/sessions/abc-123/events"),
    ).toBe(true);
  });

  it("rejects other paths", () => {
    expect(isSessionEventsURL("http://localhost:8080/sessions/abc-123")).toBe(
      false,
    );
    expect(isSessionEventsURL("http://localhost:8080/health")).toBe(false);
  });
});
