# Users sign in with email and password, handled by our own server

We chose email + password over Sign in with Apple. This means our Go server stores password hashes and owns the whole credential lifecycle, so it is security-critical code we must get right rather than delegate.

## Consequences

- Passwords are hashed with a memory-hard algorithm. They are never logged and never stored in plain text.
- Email verification and password reset use 6-digit one-time codes: fixed `123456` in development, emailed in production (ADR-0016).
- If third-party social logins are ever added, App Store rules require Sign in with Apple to be offered too.
