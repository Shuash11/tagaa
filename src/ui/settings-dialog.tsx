import React, { useState, useCallback } from "react";
import { Box, Text, useInput } from "ink";
import { appState } from "./app-state.js";

const PROVIDERS = [
  { id: "anthropic", label: "Anthropic", env: "ANTHROPIC_API_KEY" },
  { id: "openai", label: "OpenAI", env: "OPENAI_API_KEY" },
  { id: "google", label: "Google", env: "GOOGLE_API_KEY" },
  { id: "mistral", label: "Mistral", env: "MISTRAL_API_KEY" },
  { id: "deepseek", label: "DeepSeek", env: "DEEPSEEK_API_KEY" },
  { id: "xai", label: "xAI", env: "XAI_API_KEY" },
  { id: "nvidia", label: "NVIDIA", env: "NVIDIA_API_KEY" },
];

const B = "#1a1a2e";
const CW = 42;

export const SettingsDialog: React.FC = () => {
  const [selected, setSelected] = useState(0);
  const [editing, setEditing] = useState(false);
  const [input, setInput] = useState("");

  const close = useCallback(() => appState.setConfigOpen(false), []);

  useInput((ch, key) => {
    if (key.escape) {
      if (editing) { setEditing(false); setInput(""); }
      else close();
      return;
    }
    if (editing) {
      if (key.return) {
        const p = PROVIDERS[selected];
        if (input.trim()) process.env[p.env] = input.trim();
        setEditing(false);
        setInput("");
      } else if (key.backspace || key.delete) {
        setInput((v) => v.slice(0, -1));
      } else if (!key.ctrl && !key.meta && ch.length === 1) {
        setInput((v) => v + ch);
      }
      return;
    }
    if (key.return) setEditing(true);
    else if (key.upArrow || key.downArrow) {
      setSelected((i) => (i + (key.upArrow ? -1 : 1) + PROVIDERS.length) % PROVIDERS.length);
    }
  });

  return (
    <Box position="absolute" width="100%" justifyContent="center" paddingTop={3}>
      <Box flexDirection="column" borderStyle="round" borderColor="#5B8DEF" paddingX={1} paddingY={1}>
        <Text backgroundColor={B}>{" API Configuration".padEnd(CW)}{"\n"}</Text>
        <Text backgroundColor={B}>{"".padEnd(CW)}{"\n"}</Text>
        {PROVIDERS.map((p, i) => {
          const active = selected === i && !editing;
          const env = process.env[p.env] || "";
          const hasKey = env.length > 0 && env !== "${" + p.env + "}";
          const label = p.label.padEnd(12);
          const status = editing && selected === i
            ? "*".repeat(Math.min(input.length || 1, 16))
            : hasKey ? "● configured" : "○ empty";
          const line = `${active ? "▸" : " "} ${label}  ${status}`;
          return (
            <Text key={p.id} backgroundColor={B} color={active ? "#5B8DEF" : undefined}>
              {line.padEnd(CW)}
              {"\n"}
            </Text>
          );
        })}
        <Text backgroundColor={B}>{"".padEnd(CW)}{"\n"}</Text>
        <Text backgroundColor={B} dimColor>
          {(editing ? "Enter API key, Esc to cancel" : "↑↓ select, Enter to edit, Esc to close").padEnd(CW)}
          {"\n"}
        </Text>
      </Box>
    </Box>
  );
};