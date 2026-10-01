# Members are added by email address, and a verified sign-up Claims the matching Placeholder

Any Member (not only Admins) can add a person to a Group by email address. If a User already has that email, they become a Member at once and get an in-app Notification; no email is sent. Otherwise a Placeholder Member is created that carries the email. When someone later signs up and **verifies** that email, every Placeholder carrying it is Claimed automatically, and from then on they act like any other Member. Until then, they remain a Placeholder.

## Consequences

- The automatic Claim happens only after email verification. In development that is the fixed code `123456` (ADR-0016), so anyone can claim any address locally. That is acceptable only because the server refuses fixed codes outside development.
- Invite Links are built now (creation, sharing, expiry, revocation, the join API and screen) but only open the app once a domain exists for Universal Links.
- The other Claim paths (a targeted Invite Link, or picking a Placeholder with Admin confirmation) still apply.
