# Conversation logs show prompts

Follows [15](15-retro-guardrails.md). Branch `docs/log-prompts`.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-01 13:51:26Z | User | Asked for the conversation logs to show the prompts instead of details | — | update the conversation docs - for me - instead of details - place what prompts |
| 2026-10-01 13:51:43Z | Claude | Asked which logs to change and what Claude rows should hold | — | — |
| 2026-10-01 13:52:36Z | User | Chose all logs plus the convention, with `—` on Claude rows | Prompt column replaces Details everywhere | Answered: "Swap the Details column for a Prompt column holding your exact words? Which logs should change?"="All logs + convention (Recommended)", "What goes in the Prompt column on Claude's rows?"="— (Recommended)" |
| 2026-10-01 13:55:34Z | Claude | Rebuilt files 01–15 from the session transcripts: Details → Prompt with the exact words; User rows moved to their real times; added the missing prompt rows (`merge PR #4`, the #30 log PR, `merge #34`) and `00-setup-skills-and-github.md` for the first session; updated the README convention and CLAUDE.md | Old Details text stays in git history | — |
