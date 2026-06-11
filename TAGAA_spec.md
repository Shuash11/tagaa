# TAGAA — Terminal Autonomous Group AI Assistant
### Complete System Specification & Workflow Document

> **Version:** 0.1.0-draft  
> **Status:** Pre-implementation  
> **Purpose:** Blueprint for a multi-model, terminal-native autonomous group chat assistant that collaborates, plans, votes, and executes tasks with accuracy-layered consensus.

---

## Table of Contents

1. [Overview](#overview)
2. [Core Philosophy](#core-philosophy)
3. [System Architecture](#system-architecture)
4. [AI Agents & Model Roster](#ai-agents--model-roster)
5. [The Five Phases](#the-five-phases)
   - Phase 1: Problem Intake & Decomposition
   - Phase 2: Plan Generation
   - Phase 3: Plan Vote (Best Strategy)
   - Phase 4: Compatibility Vote (Best Executor)
   - Phase 5: Execution & Bug Fix Council
6. [Voting Subsystem](#voting-subsystem)
7. [Bug Fix Council Protocol](#bug-fix-council-protocol)
8. [Messenger UI / Terminal Chat Feel](#messenger-ui--terminal-chat-feel)
9. [Tool & File System Access](#tool--file-system-access)
10. [Accuracy Enhancement Layers](#accuracy-enhancement-layers)
11. [Tech Stack](#tech-stack)
12. [Project File Structure](#project-file-structure)
13. [Config File Reference](#config-file-reference)
14. [Data Flow Diagram](#data-flow-diagram)
15. [Glossary](#glossary)

---

## Overview

**TAGAA** is a terminal-based autonomous multi-agent system that feels like a group chat — but the participants are AI models from different providers. When you give them a task, they don't just execute; they **plan together, debate strategies, vote on the best approach, elect the most compatible executor for the task, and then autonomously implement the solution** — all while streaming conversation to your terminal like a messenger.

The system is built for:

- Code generation, editing, and refactoring
- Bug diagnosis and fixing
- Autonomous task execution (file I/O, shell commands, APIs)
- Research and synthesis tasks

Every model has a name, a provider, a specialty tag, and a "voice" that makes the group chat feel alive — not just a pipeline.

---

## Core Philosophy

**Consensus over speed.** A single model answering fast is good. Five models cross-examining each other's plans and voting on accuracy is better.

**Specialization matters.** Not every model is equally good at debugging, code generation, security review, or architecture planning. TAGAA routes execution to the most qualified agent based on structured votes.

**Transparency is non-negotiable.** Every vote, every score, every plan is shown to the user in the terminal. Nothing happens in a hidden pipeline.

**The group can disagree.** Dissenting opinions are surfaced, not silenced. A minority vote is displayed as a challenge that the winning agent must acknowledge.

---

## System Architecture

```
┌─────────────────────────────────────────────────────┐
│                   TAGAA Terminal UI                  │
│           (Messenger-style chat renderer)            │
└────────────────────────┬────────────────────────────┘
                         │
              ┌──────────▼──────────┐
              │    Orchestrator      │
              │  (Phase Controller)  │
              └──────────┬──────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼──────┐  ┌──────▼──────┐  ┌─────▼────────┐
│  Agent Pool  │  │ Vote Engine │  │  Exec Engine │
│  (N models)  │  │ (Scoring)   │  │  (Code/Bash) │
└───────┬──────┘  └──────┬──────┘  └─────┬────────┘
        │                │               │
        └────────────────┼───────────────┘
                         │
              ┌──────────▼──────────┐
              │    State Manager     │
              │  (Phase + History)   │
              └─────────────────────┘
```

The **Orchestrator** is the traffic controller. It does not vote, but it enforces phase transitions, collects outputs from the Agent Pool, feeds them into the Vote Engine, and hands off the winning plan + executor to the Exec Engine.

---

## AI Agents & Model Roster

Each agent has a fixed identity in the group chat. Below is the recommended default roster. The config file can override or extend this.

| Agent Name   | Provider    | API Model String                      | Specialty Tag           | Chat Color  |
|--------------|-------------|---------------------------------------|-------------------------|-------------|
| **Sonnet**   | Anthropic   | `claude-sonnet-4-5`                   | Architecture, Planning  | Cyan        |
| **Opus**     | Anthropic   | `claude-opus-4-5`                     | Deep Reasoning, Review  | Blue        |
| **Haiku**    | Anthropic   | `claude-haiku-4-5`                    | Speed, Summarization    | Green       |
| **GPT4o**    | OpenAI      | `gpt-4o`                              | General, Code Gen       | Yellow      |
| **o3**       | OpenAI      | `o3`                                  | Logical Reasoning       | Orange      |
| **Gemini**   | Google      | `gemini-2.5-pro`                      | Research, Multimodal    | Magenta     |
| **Mistral**  | Mistral AI  | `mistral-large-latest`                | Concise Answers, Speed  | White       |
| **DeepSeek** | DeepSeek    | `deepseek-coder`                      | Low-level Code, DSA     | Red         |
| **Grok**     | xAI         | `grok-3`                              | Creativity, Brainstorm  | Purple      |

> **Minimum viable roster:** 3 agents. You need at least 3 for voting to produce a meaningful majority.  
> **Maximum recommended:** 7 agents (beyond that, latency and token costs increase significantly without proportional accuracy gains).

Each agent is initialized with a **system prompt** that includes:
- Its name and specialty
- The current task context
- The current phase it is operating in
- Its voting responsibilities for that phase

---

## The Five Phases

### Phase 0 — Standby (Idle)

The terminal is open. The user can type. A single "chat" input line is shown. All agents are dormant. No API calls are made.

```
[TAGAA] Group is ready. Type a task or /help
> 
```

---

### Phase 1 — Problem Intake & Decomposition

**Trigger:** User submits a task (hits Enter).

**What happens:**

The Orchestrator receives the raw task string and does the following synchronously before involving any agent:

1. Tokenizes and classifies the task type: `code_edit`, `new_feature`, `bug_fix`, `research`, `refactor`, `shell_task`, or `general`.
2. If files are referenced, reads them from disk and appends their content to the shared context window.
3. Constructs the **Task Brief** — a structured object passed to all agents.

**Task Brief format:**
```json
{
  "task_id": "uuid-v4",
  "raw_input": "fix the race condition in worker.ts",
  "classified_type": "bug_fix",
  "attached_files": ["src/worker.ts"],
  "file_contents": { "src/worker.ts": "..." },
  "timestamp": "ISO-8601"
}
```

**Terminal output:**
```
[ORCHESTRATOR] Task received → Type: bug_fix
[ORCHESTRATOR] Files loaded: src/worker.ts (312 lines)
[ORCHESTRATOR] Briefing all agents...
```

---

### Phase 2 — Plan Generation

**What happens:**

All agents are called **in parallel** (concurrent API calls). Each agent receives the Task Brief and is prompted to return a **structured plan** in a defined schema.

**Plan schema (JSON):**
```json
{
  "agent": "Sonnet",
  "plan_id": "uuid",
  "summary": "One sentence describing the approach",
  "steps": [
    { "step": 1, "action": "description of what to do", "target_file": "optional" }
  ],
  "estimated_complexity": "low | medium | high",
  "risks": ["list of potential failure points"],
  "self_confidence": 0.0
}
```

`self_confidence` is a float 0–1 that the model provides about its own plan. This is used in tiebreaking only — it does not affect the vote.

**Terminal output (streaming, messenger-style):**
```
┌─[Sonnet]──────────────────────────────────────────────┐
│ Plan: Wrap the shared queue access with a mutex lock.  │
│ Steps: (1) Identify the critical section (2) Import    │
│ mutex primitive (3) Wrap enqueue/dequeue calls         │
│ Complexity: medium | Risks: deadlock if lock order     │
│ changes                                                │
└────────────────────────────────────────────────────────┘

┌─[GPT4o]────────────────────────────────────────────────┐
│ Plan: Replace shared queue with a lock-free MPSC       │
│ channel using Node's AsyncLocalStorage pattern.        │
│ Steps: (1) Audit current data flow (2) Swap queue...   │
└────────────────────────────────────────────────────────┘

┌─[DeepSeek]─────────────────────────────────────────────┐
│ Plan: Use a semaphore with a bounded counter...        │
└────────────────────────────────────────────────────────┘
```

Each plan is stored in the State Manager.

---

### Phase 3 — Plan Vote (Best Strategy)

**What happens:**

Every agent reads **all other agents' plans** (excluding its own) and votes for the best one. This is a **blind cross-evaluation** — agents score plans, not agents.

Each voter must provide:

```json
{
  "voter": "Haiku",
  "voted_for_plan_id": "uuid-of-sonnet-plan",
  "scores": {
    "correctness": 8,
    "safety": 9,
    "simplicity": 7,
    "completeness": 8
  },
  "reason": "Mutex approach is the most battle-tested solution for this pattern. Lock-free channels add complexity risk."
}
```

**Scoring dimensions (all 1–10):**

| Dimension     | What it measures                                               |
|---------------|----------------------------------------------------------------|
| `correctness` | Does the plan actually solve the described problem?           |
| `safety`      | Does it avoid introducing new bugs, regressions, or risks?   |
| `simplicity`  | Is the solution as simple as it can be while still working?  |
| `completeness`| Does it cover edge cases, error handling, cleanup?           |

**Vote aggregation:**

```
Final Score = (correctness × 0.35) + (safety × 0.25) + (completeness × 0.25) + (simplicity × 0.15)
```

Weights are configurable in `tagaa.config.json`. The plan with the highest aggregated weighted score wins.

**Terminal output:**
```
[VOTE: PLAN] Agents are reviewing each other's plans...

  PLAN SCORES
  ──────────────────────────────────────────────
  Sonnet's Plan     │ ████████████  8.42  ← WINNER
  GPT4o's Plan      │ ████████░░░░  7.11
  DeepSeek's Plan   │ ██████░░░░░░  6.30
  ──────────────────────────────────────────────

[Opus] "Sonnet's mutex approach wins — it's the most conservative fix
        and least likely to introduce new concurrency bugs."
[Grok] "I voted for GPT4o's channel approach — more elegant long-term,
        but I acknowledge the majority favors pragmatism here."

[ORCHESTRATOR] Winning plan: Sonnet's Plan (score: 8.42)
```

Minority opinions are **always surfaced**. The winning agent must acknowledge dissent before Phase 5.

---

### Phase 4 — Compatibility Vote (Best Executor)

**What happens:**

Now that the **winning plan** is chosen, the group votes on **which agent is most capable of implementing it** given the classified task type, the specific plan's nature, and each agent's known specialty.

This is separate from the plan vote because the best strategist is not always the best implementer.

Each agent submits a compatibility ballot:

```json
{
  "voter": "Gemini",
  "nominated_executor": "DeepSeek",
  "reasoning": "DeepSeek is the strongest at low-level TypeScript concurrency patterns. The plan involves mutex primitives which is deep systems code.",
  "specialty_match_score": 9,
  "context_fit_score": 8
}
```

**Compatibility is scored on:**

| Dimension            | What it measures                                             |
|----------------------|--------------------------------------------------------------|
| `specialty_match`    | How well the agent's core strength fits the plan's nature   |
| `context_fit`        | Has the agent demonstrated the best understanding of the attached files so far? |

An agent may nominate itself, but self-nominations are penalized by 10% to prevent bias.

**Terminal output:**
```
[VOTE: EXECUTOR] Who should implement the winning plan?

  COMPATIBILITY SCORES
  ──────────────────────────────────────────────
  DeepSeek   │ ████████████  9.10  ← SELECTED
  Sonnet     │ ████████░░░░  7.80
  GPT4o      │ ███████░░░░░  6.90
  ──────────────────────────────────────────────

[ORCHESTRATOR] Executor elected: DeepSeek
```

---

### Phase 5 — Execution & Bug Fix Council

**What happens:**

The elected executor implements the winning plan. It streams its output directly to the terminal — file edits, bash commands, code diffs — all visible in real time.

#### 5a — Execution

The executor agent receives:
- The winning plan (structured)
- All attached file contents
- Shell access (via the Tool Layer)
- A strict output schema it must follow

Execution output types:

| Output Type   | Terminal display                    |
|---------------|-------------------------------------|
| `file_edit`   | Unified diff rendered inline        |
| `bash_cmd`    | Command shown, then stdout/stderr   |
| `code_block`  | Syntax-highlighted code block       |
| `message`     | Plain agent message                 |

#### 5b — Real-time Review Panel

While the executor works, **two reviewer agents** (randomly assigned from the non-executor pool) watch the stream and can interject with:

```
[Opus] ⚠️  Line 47: that mutex scope might not cover the async callback on line 52.
```

The executor can acknowledge and patch, or explain why the reviewer is wrong. This is surfaced as real-time chat.

#### 5c — Bug Fix Council (Post-Execution)

After execution completes, the system automatically runs any available test suite (or a basic syntax check). If **errors or test failures are detected**, the Bug Fix Council is invoked.

**Council Protocol:**

1. All agents receive the error output.
2. Each agent proposes a **bug fix plan** (same schema as Phase 2).
3. The group votes using the **Bug Fix Vote** (same scoring, same dimensions as Phase 3).
4. The highest-scored fix is assigned to an executor (Phase 4 re-run, lightweight).
5. Fix is applied and tests re-run.
6. Council repeats up to `max_fix_cycles` (default: 3) before escalating to the user.

**Terminal output:**
```
[ORCHESTRATOR] Test runner detected 1 failure.
[ORCHESTRATOR] Invoking Bug Fix Council...

┌─[Sonnet]────────────────────────────────────────────┐
│ The mutex scope needs to wrap the async .then()     │
│ chain, not just the synchronous enqueue call.       │
└─────────────────────────────────────────────────────┘
┌─[GPT4o]─────────────────────────────────────────────┐
│ We should switch to using AsyncMutex from the       │
│ async-mutex package — it handles Promise chains.    │
└─────────────────────────────────────────────────────┘

  BUG FIX VOTE
  ──────────────────────────────────────
  GPT4o's Fix    │ ████████████  8.90 ← WINNER
  Sonnet's Fix   │ ████████░░░░  7.60
  ──────────────────────────────────────

[ORCHESTRATOR] Applying GPT4o's fix via DeepSeek...
[ORCHESTRATOR] Tests re-running...
[ORCHESTRATOR] ✓ All tests passed. Bug fix cycle complete.
```

---

## Voting Subsystem

All votes go through the **Vote Engine** — a standalone module with no AI involvement. It is purely deterministic.

### Vote Engine Rules

- Each agent casts exactly one vote per round.
- An agent cannot vote for its own plan (in plan vote) but can self-nominate in executor vote (with 10% penalty).
- Scores are normalized to prevent outlier inflation.
- If there is a tie (within 0.5 points), a **tiebreak round** is triggered: the two tied agents debate in one exchange and voters re-score only those two plans.
- All raw vote data is saved to `session_log.json` for transparency.

### Vote Transparency Log (per session)

Every vote is logged:

```json
{
  "phase": "plan_vote",
  "round": 1,
  "votes": [
    {
      "voter": "Haiku",
      "voted_for": "plan-uuid-sonnet",
      "scores": { "correctness": 8, "safety": 9, "simplicity": 7, "completeness": 8 },
      "reason": "..."
    }
  ],
  "result": {
    "winner_plan_id": "plan-uuid-sonnet",
    "final_score": 8.42
  }
}
```

---

## Bug Fix Council Protocol

The Bug Fix Council is the accuracy engine for error recovery. It runs automatically whenever:

- A test runner exits with a non-zero code
- A bash command produces stderr output flagged as critical
- The executor agent itself signals uncertainty (`confidence < 0.5`)

### Council Cycle

```
Error detected
     │
     ▼
All agents propose fix plans (parallel)
     │
     ▼
Fix Plan Vote (same scoring matrix)
     │
     ▼
Compatibility Vote for fix executor
     │
     ▼
Fix applied
     │
     ▼
Tests re-run
     │
   ┌─┴─────────────────┐
   │                   │
Pass                 Fail
   │                   │
Done           Cycle count < max?
                       │
                   Yes─┤─No
                       │   │
                  Loop ▼   ▼
                        Escalate to user
```

### Escalation Message

When max cycles are hit:
```
[ORCHESTRATOR] ⛔ Bug Fix Council exhausted 3 cycles without resolution.
[ORCHESTRATOR] Surfacing all proposed fixes for manual review.

  Fix 1 (Sonnet)   — Score: 7.40 — [view diff]
  Fix 2 (GPT4o)    — Score: 8.20 — [view diff]  ← Highest
  Fix 3 (DeepSeek) — Score: 6.10 — [view diff]

> Apply fix 2? (y/n/edit):
```

---

## Messenger UI / Terminal Chat Feel

The terminal UI is built to feel like a real group chat. Each agent has a persistent visual identity.

### Message Format

```
┌─[AgentName]────────────────── specialty_tag ──────── HH:MM ─┐
│ Message content goes here. It wraps if it's long and the    │
│ terminal column width is respected.                         │
└─────────────────────────────────────────────────────────────┘
```

### Special Message Types

| Type               | Visual indicator                             |
|--------------------|----------------------------------------------|
| Voting             | `[VOTE]` badge + score bar rendered in ASCII |
| Warning / Dissent  | `⚠️` prefix, yellow border                   |
| Execution output   | `>` prefix, dim color, monospace             |
| Diff output        | `+` green / `-` red lines (like git diff)    |
| Orchestrator msg   | `[ORCHESTRATOR]` label, no box, bold         |
| Error              | `⛔` prefix, red text                         |
| Success            | `✓` prefix, green text                       |

### Commands (user-facing)

| Command           | Action                                         |
|-------------------|------------------------------------------------|
| `/help`           | Show all commands                              |
| `/agents`         | List active agents and their model strings     |
| `/votes`          | Show full vote log for current session         |
| `/plan <id>`      | View a specific plan's full detail             |
| `/skip vote`      | Skip voting phases and use first plan (fast mode) |
| `/add agent`      | Add an agent mid-session                       |
| `/remove <name>`  | Remove an agent                                |
| `/reset`          | Clear session state                            |
| `/export`         | Save session log to JSON                       |
| `/replay`         | Replay last session from log                  |

---

## Tool & File System Access

Agents interact with the host system via a **Tool Layer** — a sandboxed set of functions that the executor agent can call.

### Available Tools

| Tool              | Description                                           | Allowed in phases  |
|-------------------|-------------------------------------------------------|--------------------|
| `read_file`       | Read file content from disk                           | 1, 2, 3, 4, 5      |
| `write_file`      | Write content to a file (creates or overwrites)       | 5 only             |
| `edit_file`       | Apply a diff to an existing file                      | 5 only             |
| `run_command`     | Execute a shell command, return stdout/stderr         | 5 only             |
| `list_directory`  | List files in a directory                             | 1, 5               |
| `search_codebase` | grep/ripgrep over project files                       | 1, 2, 5            |
| `run_tests`       | Run the configured test command and return results    | 5 only             |
| `fetch_url`       | HTTP GET a URL (for research tasks)                   | 2, 5               |

### Tool Call Format (internal)

Agents request tool calls in structured output:

```json
{
  "tool": "edit_file",
  "args": {
    "path": "src/worker.ts",
    "diff": "--- a/src/worker.ts\n+++ b/src/worker.ts\n@@ -44,6 +44,9 @@\n..."
  }
}
```

The Tool Layer validates the call, executes it, and returns a result to the agent before it continues generating output.

---

## Accuracy Enhancement Layers

Beyond voting, TAGAA implements additional accuracy measures at each phase.

### Layer 1 — Structured Output Enforcement

All agent outputs are validated against JSON schemas before they are accepted. Malformed outputs are rejected and the agent is re-prompted once.

### Layer 2 — Cross-Examination Round (optional)

If enabled in config, after plans are generated (Phase 2), agents can ask each other one clarifying question before voting. This surfaces hidden assumptions and improves vote quality.

```
[Haiku → GPT4o] "Your plan mentions AsyncLocalStorage — has this been tested
                  with worker threads sharing memory in Node 20?"
[GPT4o]          "Good catch. That pattern is scoped to async context, not
                  shared memory. I'll revise my plan."
```

### Layer 3 — Confidence Threshold Gating

If no plan exceeds a minimum confidence threshold (aggregated vote score < 6.0), execution is blocked and the user is prompted:

```
[ORCHESTRATOR] ⚠️  No plan reached the confidence threshold (best: 5.8 / 10.0)
[ORCHESTRATOR] Recommend providing more context or attaching related files.
> Attach file or continue anyway? (file/continue):
```

### Layer 4 — Dissent Acknowledgment Requirement

The winning executor must read all dissenting votes before executing and produce a brief written acknowledgment:

```
[DeepSeek] I note that Grok and GPT4o preferred the channel-based approach.
            I'll implement the mutex solution as voted, but I'll add a comment
            in the code noting the alternative for future maintainers.
```

### Layer 5 — Post-Execution Peer Review

After execution, two randomly selected reviewer agents perform a final read of all changed files and submit a review:

```json
{
  "reviewer": "Opus",
  "verdict": "approved | changes_requested",
  "comments": [
    { "file": "src/worker.ts", "line": 52, "note": "..." }
  ]
}
```

If both reviewers return `changes_requested`, a lightweight fix cycle is triggered.

### Layer 6 — Session Replay & Audit

All phases, plans, votes, tool calls, and outputs are written to `session_log.json`. You can replay any session deterministically using `/replay` to audit decisions.

---

## Tech Stack

### Runtime & Language

| Component         | Technology                    | Reason                                          |
|-------------------|-------------------------------|-------------------------------------------------|
| Core runtime      | **Node.js 22+**               | Async/parallel API calls, broad ecosystem       |
| Language          | **TypeScript 5.x**            | Type safety across agent schemas and tool calls |
| Package manager   | **pnpm**                      | Fast installs, workspace support                |

### Terminal UI

| Component         | Technology                    | Reason                                          |
|-------------------|-------------------------------|-------------------------------------------------|
| Terminal rendering| **Ink (React for CLIs)**       | React component model for terminal UIs          |
| Syntax highlight  | **cli-highlight**             | Inline code coloring in terminal                |
| Diff rendering    | **diff** + custom ANSI        | Colored unified diffs inline                    |
| Spinner / progress| **ora**                       | Clean loading indicators                        |
| Prompts           | **@inquirer/prompts**         | Interactive user input (y/n, select, text)      |

### AI Provider SDKs

| Provider     | SDK / Package                        |
|--------------|--------------------------------------|
| Anthropic    | `@anthropic-ai/sdk`                  |
| OpenAI       | `openai`                             |
| Google       | `@google/genai`                      |
| Mistral      | `@mistralai/mistralai`               |
| DeepSeek     | OpenAI-compatible, use `openai` SDK with custom `baseURL` |
| xAI (Grok)   | OpenAI-compatible, use `openai` SDK with custom `baseURL` |

### Validation & Schema

| Component      | Technology          | Reason                              |
|----------------|---------------------|-------------------------------------|
| Schema validation | **Zod**          | Runtime type validation for all agent outputs |
| UUID generation  | **crypto.randomUUID** (native) | No dependency needed      |

### Persistence & Logging

| Component      | Technology                | Reason                                     |
|----------------|---------------------------|--------------------------------------------|
| Session log    | **JSON flat files**       | Simple, human-readable, replayable         |
| Config         | **JSON + Zod validation** | Single config file, validated on startup   |
| Embeddings cache (optional) | **sqlite3** via `better-sqlite3` | Fast local cache for context |

### Testing

| Component   | Technology       |
|-------------|------------------|
| Unit tests  | **Vitest**       |
| E2E tests   | **Vitest + mock API responses** |

### Dev Tooling

| Component   | Technology              |
|-------------|-------------------------|
| Linting     | **ESLint + typescript-eslint** |
| Formatting  | **Prettier**            |
| Build       | **tsup**                |
| Type check  | **tsc --noEmit**        |

---

## Project File Structure

```
tagaa/
├── src/
│   ├── agents/
│   │   ├── agent.ts              # Base Agent class
│   │   ├── pool.ts               # Agent pool manager (parallel calls)
│   │   └── registry.ts           # Default agent roster + config loader
│   │
│   ├── phases/
│   │   ├── phase0-standby.ts
│   │   ├── phase1-intake.ts
│   │   ├── phase2-plan.ts
│   │   ├── phase3-plan-vote.ts
│   │   ├── phase4-compat-vote.ts
│   │   └── phase5-execute.ts
│   │
│   ├── voting/
│   │   ├── vote-engine.ts        # Aggregation, tiebreaking, logging
│   │   ├── schemas.ts            # Zod schemas for vote payloads
│   │   └── weights.ts            # Configurable scoring weights
│   │
│   ├── bugfix/
│   │   └── council.ts            # Bug Fix Council orchestration
│   │
│   ├── tools/
│   │   ├── tool-layer.ts         # Sandboxed tool dispatcher
│   │   ├── read-file.ts
│   │   ├── write-file.ts
│   │   ├── edit-file.ts
│   │   ├── run-command.ts
│   │   ├── run-tests.ts
│   │   └── search-codebase.ts
│   │
│   ├── ui/
│   │   ├── app.tsx               # Root Ink component
│   │   ├── chat-message.tsx      # Agent message bubble
│   │   ├── vote-display.tsx      # ASCII score bars
│   │   ├── diff-viewer.tsx       # Colored diff output
│   │   └── status-bar.tsx        # Phase indicator, agent status
│   │
│   ├── orchestrator/
│   │   ├── orchestrator.ts       # Phase controller, state machine
│   │   └── state.ts              # Session state type definitions
│   │
│   ├── providers/
│   │   ├── anthropic.ts
│   │   ├── openai.ts
│   │   ├── google.ts
│   │   ├── mistral.ts
│   │   └── compat-openai.ts     # For DeepSeek, xAI (OpenAI-compat)
│   │
│   ├── logger/
│   │   └── session-logger.ts    # session_log.json writer
│   │
│   └── index.ts                 # Entrypoint
│
├── sessions/                    # Auto-created, stores session logs
├── tagaa.config.json            # User configuration
├── package.json
├── tsconfig.json
└── README.md
```

---

## Config File Reference

```json
{
  "agents": [
    {
      "name": "Sonnet",
      "provider": "anthropic",
      "model": "claude-sonnet-4-5",
      "specialty": "Architecture, Planning",
      "color": "cyan",
      "enabled": true
    },
    {
      "name": "GPT4o",
      "provider": "openai",
      "model": "gpt-4o",
      "specialty": "General, Code Gen",
      "color": "yellow",
      "enabled": true
    },
    {
      "name": "DeepSeek",
      "provider": "deepseek",
      "model": "deepseek-coder",
      "specialty": "Low-level Code, DSA",
      "color": "red",
      "enabled": true,
      "base_url": "https://api.deepseek.com/v1"
    }
  ],

  "api_keys": {
    "anthropic": "${ANTHROPIC_API_KEY}",
    "openai": "${OPENAI_API_KEY}",
    "google": "${GOOGLE_API_KEY}",
    "mistral": "${MISTRAL_API_KEY}",
    "deepseek": "${DEEPSEEK_API_KEY}",
    "xai": "${XAI_API_KEY}"
  },

  "voting": {
    "weights": {
      "correctness": 0.35,
      "safety": 0.25,
      "completeness": 0.25,
      "simplicity": 0.15
    },
    "confidence_threshold": 6.0,
    "tie_margin": 0.5,
    "self_nomination_penalty": 0.10
  },

  "execution": {
    "max_fix_cycles": 3,
    "test_command": "npm test",
    "allowed_tools": ["read_file", "write_file", "edit_file", "run_command", "run_tests"],
    "working_directory": "."
  },

  "features": {
    "cross_examination_round": true,
    "post_execution_peer_review": true,
    "dissent_acknowledgment": true,
    "session_logging": true,
    "fast_mode": false
  },

  "ui": {
    "message_width": 80,
    "show_timestamps": true,
    "show_specialty_tags": true,
    "syntax_highlighting": true
  }
}
```

---

## Data Flow Diagram

```
User Input
    │
    ▼
Phase 1: Intake
  ├─ Classify task type
  ├─ Load attached files
  └─ Build Task Brief
         │
         ▼
Phase 2: Plan Generation
  ├─ Agent A → Plan A ─┐
  ├─ Agent B → Plan B ─┼─ All parallel
  └─ Agent C → Plan C ─┘
         │
         ▼
Phase 3: Plan Vote
  ├─ Each agent scores all other plans
  ├─ Vote Engine aggregates (weighted)
  ├─ Tiebreak if needed
  └─ Winning Plan selected
         │
         ▼
Phase 4: Compatibility Vote
  ├─ Each agent nominates best executor
  ├─ Vote Engine aggregates
  └─ Executor agent selected
         │
         ▼
Phase 5: Execution
  ├─ Executor reads winning plan
  ├─ Executor acknowledges dissent
  ├─ Executor applies changes via Tool Layer
  ├─ Reviewer agents watch stream
  └─ Test runner invoked
         │
    ┌────┴────┐
  Pass      Fail
    │         │
    ▼         ▼
  Done    Bug Fix Council
            ├─ All agents propose fixes
            ├─ Fix Vote
            ├─ Fix Executor selected
            ├─ Fix applied
            └─ Re-test (loop ≤ max_cycles)
                   │
              ┌────┴────┐
            Pass      Fail
              │         │
              ▼         ▼
            Done     Escalate to user
```

---

## Glossary

| Term                    | Definition                                                              |
|-------------------------|-------------------------------------------------------------------------|
| **Orchestrator**        | The non-AI phase controller that drives the workflow state machine      |
| **Agent**               | An AI model with a name, provider, model string, and specialty          |
| **Agent Pool**          | The set of all active agents in a session                               |
| **Task Brief**          | The structured input object shared with all agents after Phase 1        |
| **Plan**                | A structured multi-step strategy produced by an agent in Phase 2        |
| **Vote Engine**         | The deterministic scoring and aggregation module (no AI)                |
| **Executor**            | The agent elected in Phase 4 to implement the winning plan              |
| **Bug Fix Council**     | The error-recovery loop that runs when execution produces failures      |
| **Tool Layer**          | The sandboxed set of file/shell capabilities agents can invoke          |
| **Cross-Examination**   | Optional phase where agents question each other's plans before voting   |
| **Dissent Acknowledgment** | Required response from executor addressing minority vote opinions    |
| **Peer Review**         | Post-execution read by two reviewer agents of all changed files         |
| **Session Log**         | Full JSON record of all phases, votes, tool calls, and outputs          |
| **Confidence Threshold**| Minimum aggregated vote score required to proceed to execution          |
| **Fast Mode**           | Config flag that skips voting phases and uses the first plan generated  |
| **max_fix_cycles**      | Maximum number of Bug Fix Council cycles before escalating to the user  |

---

*TAGAA Specification — End of Document*  
*Built for builders who want their AIs to earn their answers.*
