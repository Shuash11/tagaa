type Listener = () => void;

export type MessageType = "agent" | "orchestrator" | "system" | "error" | "success" | "warning" | "dim" | "vote" | "divider";

export interface ChatMessage {
  id: string;
  type: MessageType;
  agentName?: string;
  color?: string;
  specialty?: string;
  content: string;
  timestamp: Date;
}

let msgCounter = 0;

export function createMessage(
  type: MessageType,
  content: string,
  opts?: { agentName?: string; color?: string; specialty?: string }
): ChatMessage {
  return {
    id: `msg-${++msgCounter}`,
    type,
    content,
    agentName: opts?.agentName,
    color: opts?.color,
    specialty: opts?.specialty,
    timestamp: new Date(),
  };
}

export class MessageStore {
  private messages: ChatMessage[] = [];
  private listeners: Set<Listener> = new Set();
  private logEnabled = true;

  constructor(logToConsole = true) {
    this.logEnabled = logToConsole;
  }

  push(msg: ChatMessage): void {
    this.messages.push(msg);
    if (this.logEnabled) {
      this.consoleLog(msg);
    }
    this.notify();
  }

  pushMany(msgs: ChatMessage[]): void {
    for (const m of msgs) {
      this.messages.push(m);
      if (this.logEnabled) this.consoleLog(m);
    }
    this.notify();
  }

  getAll(): ChatMessage[] {
    return this.messages;
  }

  disableConsoleLog(): void {
    this.logEnabled = false;
  }

  clear(): void {
    this.messages = [];
    this.notify();
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => { this.listeners.delete(listener); };
  }

  private notify(): void {
    for (const fn of this.listeners) fn();
  }

  private consoleLog(msg: ChatMessage): void {
    switch (msg.type) {
      case "agent":
        console.log(
          `[${msg.agentName}] ${msg.content.split("\n")[0]}`
        );
        break;
      case "orchestrator":
        console.log(`[ORCHESTRATOR] ${msg.content}`);
        break;
      case "error":
        console.error(`⛔ ${msg.content}`);
        break;
      case "success":
        console.log(`✓ ${msg.content}`);
        break;
      case "warning":
        console.log(`⚠ ${msg.content}`);
        break;
      case "dim":
        console.log(`  ${msg.content}`);
        break;
      case "vote":
        console.log(msg.content);
        break;
      case "divider":
        console.log("");
        break;
      default:
        console.log(msg.content);
    }
  }
}

export const messageStore = new MessageStore(true);