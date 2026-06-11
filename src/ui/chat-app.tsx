import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import {
  AgentMessage,
  OrchestratorMsg,
  SuccessMsg,
  ErrorMsg,
  WarningMsg,
  DimMsg,
  Divider,
} from "./agent-bubble.js";
import { messageStore, ChatMessage } from "./message-store.js";
import { appState, AgentInfo } from "./app-state.js";
import { Sidebar } from "./sidebar.js";
import { SettingsDialog } from "./settings-dialog.js";

const B = "#16161e";

interface ChatAppProps {
  onInput: (input: string) => void;
  agents?: AgentInfo[];
}

export const ChatApp: React.FC<ChatAppProps> = ({ onInput, agents = [] }) => {
  const [messages, setMessages] = useState<ChatMessage[]>(() => messageStore.getAll());
  const [inputValue, setInputValue] = useState("");
  const [sidebarVisible, setSidebarVisible] = useState(() => appState.sidebarVisible);
  const [configOpen, setConfigOpen] = useState(false);

  useEffect(() => {
    if (agents.length > 0) {
      appState.setAgents(agents);
      appState.setApiKeys(
        agents.map((a) => ({ provider: a.provider, configured: true }))
      );
    }
  }, [agents]);

  useEffect(() => {
    const u1 = appState.subscribe(() => {
      setSidebarVisible(appState.sidebarVisible);
      setConfigOpen(appState.configOpen);
    });
    const u2 = messageStore.subscribe(() => {
      setMessages([...messageStore.getAll()]);
    });
    return () => { u1(); u2(); };
  }, []);

  useInput((ch, key) => {
    if (configOpen) return;

    if (key.ctrl && ch === "b") { appState.toggleSidebar(); return; }
    if (key.ctrl && ch === "s") { appState.setConfigOpen(true); return; }
    if (key.return && inputValue.trim()) {
      appState.setRunning(true);
      appState.setPhase("intake");
      onInput(inputValue.trim());
      setInputValue("");
      return;
    }
    if (key.backspace || key.delete) { setInputValue((v) => v.slice(0, -1)); return; }
    if (!key.ctrl && !key.meta && ch.length === 1) { setInputValue((v) => v + ch); }
  });

  const renderMessage = (msg: ChatMessage) => {
    switch (msg.type) {
      case "agent":
        return <AgentMessage key={msg.id} name={msg.agentName || ""} color={msg.color || "white"} specialty={msg.specialty} content={msg.content} />;
      case "orchestrator": return <OrchestratorMsg key={msg.id} content={msg.content} />;
      case "success": return <SuccessMsg key={msg.id} content={msg.content} />;
      case "error": return <ErrorMsg key={msg.id} content={msg.content} />;
      case "warning": return <WarningMsg key={msg.id} content={msg.content} />;
      case "dim": return <DimMsg key={msg.id} content={msg.content} />;
      case "divider": return <Divider key={msg.id} />;
      default: return <DimMsg key={msg.id} content={msg.content} />;
    }
  };

  return (
    <Box position="relative" flexDirection="row" width="100%">
      <Box flexDirection="column" flexGrow={1}>
        <Box paddingX={1} borderStyle="single" borderColor="#2c313a">
          <Text backgroundColor={B}>
            <Text bold color="#00CED1"> TAGAA  </Text>
            <Text dimColor>Terminal Autonomous Group AI Assistant  </Text>
            <Text dimColor>v0.1.0</Text>
          </Text>
        </Box>

        <Box flexDirection="column" flexGrow={1} paddingX={1} paddingTop={1}>
          <Text backgroundColor={B}>
            {messages.length === 0 ? "  Waiting for input...".padEnd(60) : ""}
            {"\n"}
          </Text>
          {messages.map(renderMessage)}
        </Box>

        <Box paddingX={1}>
          <Text backgroundColor={B} bold color="#00CED1">λ </Text>
          <Text backgroundColor={B}>{inputValue}</Text>
        </Box>
      </Box>

      {sidebarVisible && <Sidebar />}
      {configOpen && <SettingsDialog />}
    </Box>
  );
};