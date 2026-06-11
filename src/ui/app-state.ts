export type AppPhase = "idle" | "intake" | "plan" | "vote" | "execute" | "done" | "error";

export interface AgentInfo {
  name: string;
  provider: string;
  model: string;
  specialty: string;
  color: string;
  enabled: boolean;
}

export interface ApiKeyStatus {
  provider: string;
  configured: boolean;
}

type Listener = () => void;

class AppStateStore {
  private _sidebarVisible = true;
  private _phase: AppPhase = "idle";
  private _agents: AgentInfo[] = [];
  private _apiKeys: ApiKeyStatus[] = [];
  private _isRunning = false;
  private _configOpen = false;
  private listeners: Set<Listener> = new Set();

  get sidebarVisible() { return this._sidebarVisible; }
  get phase() { return this._phase; }
  get agents() { return this._agents; }
  get apiKeys() { return this._apiKeys; }
  get isRunning() { return this._isRunning; }
  get configOpen() { return this._configOpen; }

  toggleSidebar() {
    this._sidebarVisible = !this._sidebarVisible;
    this.notify();
  }

  setSidebar(v: boolean) {
    this._sidebarVisible = v;
    this.notify();
  }

  setPhase(p: AppPhase) {
    this._phase = p;
    this.notify();
  }

  setAgents(a: AgentInfo[]) {
    this._agents = a;
    this.notify();
  }

  setApiKeys(k: ApiKeyStatus[]) {
    this._apiKeys = k;
    this.notify();
  }

  setRunning(r: boolean) {
    this._isRunning = r;
    this.notify();
  }

  setConfigOpen(v: boolean) {
    this._configOpen = v;
    this.notify();
  }

  subscribe(l: Listener): () => void {
    this.listeners.add(l);
    return () => { this.listeners.delete(l); };
  }

  private notify() {
    for (const fn of this.listeners) fn();
  }
}

export const appState = new AppStateStore();
