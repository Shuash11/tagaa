import { describe, it, expect } from "vitest";
import { createSessionState, SessionState } from "../orchestrator/state.js";

describe("createSessionState", () => {
  it("should create a session in standby phase", () => {
    const state = createSessionState("test-session-1");
    expect(state.sessionId).toBe("test-session-1");
    expect(state.phase).toBe("standby");
    expect(state.plans.size).toBe(0);
    expect(state.bugFixCycles).toBe(0);
    expect(state.fastMode).toBe(false);
  });

  it("should have unique session IDs", () => {
    const state1 = createSessionState("a");
    const state2 = createSessionState("b");
    expect(state1.sessionId).not.toBe(state2.sessionId);
  });
});