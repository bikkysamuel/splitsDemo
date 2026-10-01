# Conversation log

A chronological record of the conversations that shaped this project. The decision log is `docs/00-open-questions.md`, and ADRs and `GLOSSARY.md` hold the decisions themselves. These files record *how we got there*.

**Just the conversation:** [`conversation-in-order.md`](conversation-in-order.md) lists every prompt and every question Claude asked, oldest first. It is generated from the session transcripts; run `scripts/conversation-in-order.py` to refresh it.

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
| Prompt | On User rows, the user's exact words (newlines as `<br>`, answers to Claude's questions as `Answered: …`); `—` on Claude rows |

## Reading in order

File names sort chronologically: list `docs/conversations/*/` and read top to bottom. `00-` is the first session, logged afterwards from its transcript. There is no hand-kept index, so parallel branches never conflict over one.
