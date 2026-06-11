import { VotingConfig } from "./schemas.js";

export const DEFAULT_VOTING_CONFIG: VotingConfig = {
  weights: {
    correctness: 0.35,
    safety: 0.25,
    completeness: 0.25,
    simplicity: 0.15,
  },
  confidence_threshold: 6.0,
  tie_margin: 0.5,
  self_nomination_penalty: 0.1,
};

export function validateWeights(weights: VotingConfig["weights"]): boolean {
  const sum =
    weights.correctness +
    weights.safety +
    weights.completeness +
    weights.simplicity;
  return Math.abs(sum - 1.0) < 0.001;
}

export function createVotingConfig(overrides?: Partial<VotingConfig>): VotingConfig {
  return {
    ...DEFAULT_VOTING_CONFIG,
    ...overrides,
    weights: {
      ...DEFAULT_VOTING_CONFIG.weights,
      ...overrides?.weights,
    },
  };
}