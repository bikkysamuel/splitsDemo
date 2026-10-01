# Conversation log setup

| UTC | Who | What happened | Decided | Details |
|---|---|---|---|---|
| 2026-10-01 09:29:01Z | User | Asked for the conversation to be captured in Markdown files, organised chronologically | One dir per date, one file per task, 5-column table | Columns: UTC date-time, who, one-line summary, short decision, details. New file when the date or the task changes. |
| 2026-10-01 09:29:14Z | Claude | Created `docs/conversations/` with README (convention + index) and files 01–04 backfilled from the session transcript | Dates are UTC; a resumed task gets a new `-cont` file; logging rule added to CLAUDE.md and memory | Timestamps backfilled from the Claude Code session log. |
