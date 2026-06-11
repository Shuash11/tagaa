import React, { useState, useEffect } from "react";
import { Box, Text } from "ink";
import { appState, AgentInfo, ApiKeyStatus } from "./app-state.js";

const B = "#16161e";
const PW = 18;

export const Sidebar: React.FC = () => {
  const [agents, setAgents] = useState<AgentInfo[]>(() => appState.agents);
  const [phase, setPhase] = useState(() => appState.phase);
  const [apiKeys, setApiKeys] = useState<ApiKeyStatus[]>(() => appState.apiKeys);
  const [isRunning, setIsRunning] = useState(() => appState.isRunning);

  useEffect(() => {
    const u = appState.subscribe(() => {
      setAgents([...appState.agents]);
      setPhase(appState.phase);
      setApiKeys([...appState.apiKeys]);
      setIsRunning(appState.isRunning);
    });
    return () => u();
  }, []);

  const cfgCount = apiKeys.filter((k) => k.configured).length;

  return (
    <Box
      width={22}
      flexDirection="column"
      borderStyle="single"
      borderLeft
      borderColor="#2c313a"
      paddingLeft={1}
    >
      <Text backgroundColor={B}>
        <Text bold color="#00CED1"> TAGAA{" ".repeat(PW - 6)}</Text>
        {"\n"}
        {" ".repeat(PW)}
        {"\n"}
        <Text dimColor> STATUS{" ".repeat(PW - 7)}</Text>
        {"\n"}
        {" "}{isRunning ? <Text color="#4CCD6B">●</Text> : <Text color="#5c6370">○</Text>}{" "}{isRunning ? "Running" : "Idle"}{" ".repeat(PW - 10)}
        {"\n"}
        {"  "}{phase}{" ".repeat(Math.max(0, PW - 2 - phase.length))}
        {"\n"}
        <Text dimColor> AGENTS{" ".repeat(PW - 7)}</Text>
        {"\n"}
        {agents.map((a, i) => (
          <Text key={a.name}>
            {"  "}{a.name}{" ".repeat(Math.max(0, PW - 2 - a.name.length))}
            {i < agents.length - 1 ? "\n" : ""}
          </Text>
        ))}
        {"\n"}
        <Text dimColor> API KEYS{" ".repeat(PW - 9)}</Text>
        {"\n"}
        {"  "}{cfgCount > 0 ? <Text color="#4CCD6B">●</Text> : <Text color="#5c6370">○</Text>}{" "}{cfgCount}/{apiKeys.length}{" ".repeat(Math.max(0, PW - 8))}
        {"\n"}
        <Text dimColor> KEYS{" ".repeat(PW - 5)}</Text>
        {"\n"}
        <Text color="#5B8DEF"> Ctrl+S Setup{" ".repeat(Math.max(0, PW - 13))}</Text>
        {"\n"}
        <Text color="#5c6370"> Ctrl+B Sidebar{" ".repeat(Math.max(0, PW - 15))}</Text>
        {"\n"}
      </Text>
    </Box>
  );
};