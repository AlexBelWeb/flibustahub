const GITHUB_REPO = 'AlexBelWeb/flibustahub'
const GITHUB_NEW_ISSUE_LIMIT = 8000

export function githubIssueURL(title: string, body: string): { url: string; fits: boolean } {
  const base = `https://github.com/${GITHUB_REPO}/issues/new`
  const full = `${base}?${new URLSearchParams({ title, body }).toString()}`
  if (full.length <= GITHUB_NEW_ISSUE_LIMIT) {
    return { url: full, fits: true }
  }
  return { url: `${base}?${new URLSearchParams({ title }).toString()}`, fits: false }
}

export const SUPPORT_EMAIL = 'alexbelweb@gmail.com'

export function mailtoURL(title: string, body: string): string {
  return `mailto:${SUPPORT_EMAIL}?${new URLSearchParams({ subject: title, body }).toString()}`
}
