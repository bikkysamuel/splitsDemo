# Conversation log

A chronological record of the conversations that shaped this project. The decision log is `docs/00-open-questions.md`, and ADRs and `GLOSSARY.md` hold the decisions themselves. These files record *how we got there*.

## Convention

- One directory per **UTC date**: `YYYY-MM-DD/`.
- Inside it, one file per **task**, numbered in the order the task started: `NN-<task-slug>.md`.
- Start a new file when the UTC date changes or the task changes. If a task resumes after another task, give the resumed part a new file with `-cont` in its name, so reading files in order is reading the conversation in order.
- Each file is a table with these columns:

| Column | Content |
|---|---|
| UTC | Date and time of the message (`YYYY-MM-DD HH:MM:SSZ`) |
| Who | `User` or `Claude` |
| What happened | One line |
| Decided | A short summary of what was decided, or `—` |
| Details | The fuller substance of the exchange |

## Index

### 2026-10-01

1. [Workflow routing (/ask-matt)](2026-10-01/01-ask-matt-workflow-routing.md)
2. [Wayfinder charting, abandoned (/wayfinder)](2026-10-01/02-wayfinder-charting.md)
3. [Requirements discovery, Rounds 1–4 (/grill-with-docs)](2026-10-01/03-grill-with-docs-discovery.md)
4. [Conversation log setup](2026-10-01/04-conversation-log-setup.md)
5. [Requirements discovery, Round 5 onward](2026-10-01/05-grill-with-docs-discovery-cont.md)
6. [Writing docs 01–10 and CLAUDE.md](2026-10-01/06-sdlc-docs-authoring.md)
7. [Writing the v1 spec (/to-spec)](2026-10-01/07-to-spec.md)
