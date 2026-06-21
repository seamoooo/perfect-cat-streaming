import { describe, expect, it } from "vitest";
import { chaosDirective } from "./chaos";

describe("chaosDirective", () => {
  it("returns null for empty / undefined / non-keyword prose", () => {
    expect(chaosDirective("")).toBeNull();
    expect(chaosDirective(undefined)).toBeNull();
    expect(chaosDirective(null)).toBeNull();
    expect(chaosDirective("ふつうの猫の動画です")).toBeNull();
  });

  it("matches each keyword as a standalone token, case-insensitively", () => {
    expect(chaosDirective("throughput demo sre")).toBe("sre");
    expect(chaosDirective("これは SRE のデモ")).toBe("sre");
    expect(chaosDirective("player error demo")).toBe("player");
    expect(chaosDirective("browser frontend crash")).toBe("frontend");
    expect(chaosDirective("api backend 500")).toBe("backend");
    expect(chaosDirective("FrOnTeNd")).toBe("frontend");
  });

  it("does not match keywords embedded inside longer words", () => {
    expect(chaosDirective("presretired absbackendx")).toBeNull();
  });

  it("is deterministic when several keywords appear (priority order)", () => {
    expect(chaosDirective("backend and sre both")).toBe("sre");
  });
});
