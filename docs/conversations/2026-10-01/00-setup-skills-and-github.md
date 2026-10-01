# Setting up agent skills and GitHub (/setup-matt-pocock-skills)

The first session, before [01](01-ask-matt-workflow-routing.md). Written afterwards from the session transcript.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 08:04:38Z | User | Ran `/setup-matt-pocock-skills` | — | `/setup-matt-pocock-skills` |
| 2026-10-01 08:05:53Z | User | Chose GitHub Issues, the default triage labels and a new CLAUDE.md | Issue tracker: GitHub; five default triage labels; CLAUDE.md | Answered: "Where should issues for this repo live? (No git remote exists yet.)"="GitHub", "Do you want to keep the default triage labels (needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix)?"="Yes (Recommended)", "Neither CLAUDE.md nor AGENTS.md exists. Which should I create for the Agent skills block?"="CLAUDE.md" |
| 2026-10-01 08:06:00Z | Claude | Showed the planned CLAUDE.md `## Agent skills` block and `docs/agents/` files for confirmation | — | — |
| 2026-10-01 08:06:28Z | User | Confirmed | — | yes, go ahead |
| 2026-10-01 08:06:40Z | Claude | Wrote CLAUDE.md (Agent skills) and `docs/agents/issue-tracker.md`, `triage-labels.md`, `domain.md` | — | — |
| 2026-10-01 08:12:26Z | User | Asked to connect the repo to GitHub | — | Set it up with Github |
| 2026-10-01 08:13:15Z | Claude | Found the git repo and `bikkysamuel/splitsDemo` remote already set up; installed `gh`; asked the user to run `gh auth login` | — | — |
| 2026-10-01 08:15:54Z | User | Asked to ignore `.idea` and push the first commit | — | ignore .idea and push the initial commit |
| 2026-10-01 08:16:18Z | Claude | Added `.gitignore` (`.idea/`) and pushed the first commit `823c268` to `master` | — | — |
| 2026-10-01 08:19:44Z | User | Reported `gh` login done | — | gh login done |
| 2026-10-01 08:20:10Z | Claude | Checked `gh` auth and Issues; created the four missing triage labels (`wontfix` existed) | GitHub ready as the issue tracker | — |
