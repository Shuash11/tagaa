import { SessionState } from "../orchestrator/state.js";

export async function runPhase0Standby(state: SessionState): Promise<void> {
  state.phase = "standby";
  console.log("[TAGAA] Group is ready. Type a task or /help");
}