# Development uses the fixed one-time code 123456; production sends real codes by email

Email verification (sign-up) and password reset both use a 6-digit one-time code. Until an email provider is chosen, the server runs with `APP_ENV=development` and an OTP sender that accepts `123456` for every account and sends nothing. In production, an email-backed sender generates random codes and emails them (provider to be chosen later). We chose this over dropping verification and reset for now, so the real flows, screens and tests exist from day one and only the delivery changes later.

## Consequences

- **Safety guard:** the server refuses to start if the fixed-code sender is configured while `APP_ENV` is anything other than `development`, and the production build must have an email sender configured. A fixed code reaching production would let anyone take over any account.
- The rest of the code rules are unchanged: 15-minute expiry, at most 5 attempts, no leaking whether an account exists, and all sessions revoked on reset.
