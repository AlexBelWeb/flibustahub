export type SecretKind = 'os' | 'file' | string

export type AIProviderId = 'gemini' | 'openai' | 'ollama'

export interface SecretStatus {
  kind: SecretKind
  machineIdMissing: boolean
  hasSecret: boolean
}

export const AI_PROVIDERS: AIProviderId[] = ['gemini', 'openai', 'ollama']
