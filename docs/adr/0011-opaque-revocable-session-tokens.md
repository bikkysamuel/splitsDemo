# Sessions use opaque, revocable tokens stored hashed in Postgres, not JWTs

The app gets a 15-minute access token and a 30-day refresh token. Both are random opaque strings whose hashes are stored in Postgres. Each refresh token is replaced when used, and reusing an old one revokes the whole session as a sign of theft. Tokens live only in the iOS Keychain. We chose this over stateless JWTs because sign-out, password reset and account deletion must revoke access immediately, and at our scale one indexed lookup per request costs nothing.
