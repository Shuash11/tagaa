import React from "react";
import { Box, Text } from "ink";

const COLORS: Record<string, string> = {
  cyan: "#00CED1",
  blue: "#5B8DEF",
  green: "#4CCD6B",
  yellow: "#E5C07B",
  orange: "#FFA500",
  magenta: "#C678DD",
  white: "#ABB2BF",
  red: "#E06C75",
  purple: "#B39DF3",
};

function hex(c: string): string {
  return COLORS[c?.toLowerCase()] || "#ABB2BF";
}

const theme = {
  bg: "#16161e",
  border: "#3E4452",
  text: "#E6E6E6",
  muted: "#5c6370",
  accent: "#00CED1",
  success: "#4CCD6B",
  error: "#E06C75",
  warning: "#E5C07B",
};

export const AgentMessage: React.FC<{
  name: string;
  color: string;
  specialty?: string;
  content: string;
}> = ({ name, color, specialty, content }) => {
  const c = hex(color);
  return (
    <Box flexDirection="column" marginBottom={1}>
      <Box>
        <Text bold color={c}>{name}</Text>
        {specialty ? <Text dimColor color={theme.muted}>{`  ${specialty}`}</Text> : null}
      </Box>
      <Text>{content}</Text>
    </Box>
  );
};

export const OrchestratorMsg: React.FC<{ content: string }> = ({ content }) => (
  <Box marginBottom={1}>
    <Text color={theme.accent}>◆ </Text>
    <Text>{content}</Text>
  </Box>
);

export const SuccessMsg: React.FC<{ content: string }> = ({ content }) => (
  <Box marginBottom={1}>
    <Text color={theme.success}>✓ </Text>
    <Text>{content}</Text>
  </Box>
);

export const ErrorMsg: React.FC<{ content: string }> = ({ content }) => (
  <Box marginBottom={1}>
    <Text color={theme.error}>✗ </Text>
    <Text>{content}</Text>
  </Box>
);

export const WarningMsg: React.FC<{ content: string }> = ({ content }) => (
  <Box marginBottom={1}>
    <Text color={theme.warning}>▲ </Text>
    <Text>{content}</Text>
  </Box>
);

export const DimMsg: React.FC<{ content: string }> = ({ content }) => (
  <Box marginLeft={3} marginBottom={1}>
    <Text dimColor>{content}</Text>
  </Box>
);

export const Divider: React.FC = () => (
  <Box marginBottom={1}>
    <Text dimColor>{`\u2500`.repeat(40)}</Text>
  </Box>
);