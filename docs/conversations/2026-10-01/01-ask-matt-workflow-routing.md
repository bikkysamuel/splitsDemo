# Workflow routing (/ask-matt)

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 08:32:14Z | User | Ran `/ask-matt`: asked Claude to act as principal architect for an expense-sharing iOS app with a Go backend | — | Production-quality iOS app inspired by expense-sharing apps; backend server in Go to handle the app's requests. |
| 2026-10-01 08:32:30Z | Claude | Recommended `/wayfinder` as the entry flow | Suggested route: `/wayfinder` → `/to-spec` → `/to-tickets` → `/implement` → `/retro` | Reasoning: greenfield product across two codebases plus an API contract, too big for one session. Listed the decision areas (domain, repo layout, API contract, auth, data, offline/sync, infra). Noted the skills setup in `docs/agents/` already looked complete. |
