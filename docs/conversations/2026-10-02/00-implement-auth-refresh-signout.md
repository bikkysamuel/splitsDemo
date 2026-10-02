# Implementing #11: silent refresh, reuse detection, session expiry, sign out, login throttling (/implement)

Follows [21](../2026-10-01/21-implement-auth-signup-signin.md). Branch `feat/auth-refresh-signout`, stacked on `feat/auth-signup-signin` (PR #40) because #11 is blocked by #10.

| UTC | Who | What happened | Decided | Prompt |
|---|---|---|---|---|
| 2026-10-02 03:09:00Z | User | Ran `/implement #11` | — | `/implement #11` |
| 2026-10-02 03:10:00Z | Claude | Read #11; #10 is still open as PR #40 (CI green), so branched from it instead of `master` | Stack #11 on #10's branch; its PR targets `feat/auth-signup-signin` until #40 merges | — |
| 2026-10-02 03:17:57Z | User | Asked to implement, open PRs and carry on through every remaining ticket without further input | Run the M1 tickets one after another unattended | implement and do pr and all till last task without my inputs<br>this time do it automatically for all the tasks until done |
| 2026-10-02 03:17:57Z | Claude | Tried to squash-merge PR #40 (per the M1 run's "merge as needed"); the auto-mode permission check refused the merge | No merges without the user: each ticket's branch and PR stack on the previous one | — |
| 2026-10-02 03:17:57Z | Claude | Server side of #11 test-first: `POST /v1/auth/refresh` (rotation within a Session family; reuse revokes the family), `POST /v1/auth/signout` (Idempotency-Key, works unverified), login throttling per account (hashed email) and per IP (1 s doubling to 15 min after 5 failures, cleared by success or an hour without failures, 429 `too-many-attempts` with `Retry-After`); `apptest` gained a second client IP on ::1 | Throttling parameters proposed (recorded as Q90); a success clears the account's count, not the IP's | — |
| 2026-10-02 03:23:05Z | Claude | iOS side of #11 test-first: `AuthenticationMiddleware` refreshes once on 401 and retries (body buffered), `SessionRefresher` shares one refresh across simultaneous 401s and reports expiry once, `AppSession` shows one "Session expired" alert, Settings shows the account and Sign out; recorded Q90 and updated doc 06 | Sign-out forgets the tokens even when the server can't be reached; a 5xx on refresh keeps the Session | — |
