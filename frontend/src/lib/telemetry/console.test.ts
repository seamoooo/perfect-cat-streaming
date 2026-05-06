import { describe, it, expect, vi } from "vitest";
import { ConsoleKanpachiSink } from "./console";

describe("ConsoleKanpachiSink", () => {
  it("logs the event name and attrs through console.log", () => {
    const spy = vi.spyOn(console, "log").mockImplementation(() => {});
    const sink = new ConsoleKanpachiSink();
    sink.record("kanpachi.first_pounce", {
      videoId: "v1",
      catName: "Kanpachi",
      breed: "bengal",
      title: "test",
      sessionId: "s1",
      playerVersion: "test",
      userAgent: "test",
      ttffMs: 123,
    });
    expect(spy).toHaveBeenCalled();
    const args = spy.mock.calls[0];
    expect(String(args[1])).toContain("kanpachi.first_pounce");
    spy.mockRestore();
  });
});
