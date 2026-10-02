/// Sign-up, email verification and sign-in (FR-A1–A4). Implementations keep
/// the Session's tokens in the Keychain; callers never see them (ADR-0005).
public protocol AuthRepository: Sendable {
  /// Creates an unverified User and signs them in, so the app can show the
  /// verification screen. A verification code is sent to `email`.
  func signUp(email: String, password: String) async throws(ServiceError) -> User
  /// Checks the code sent to `email` and signs the now verified User in.
  func verifyEmail(email: String, code: String) async throws(ServiceError) -> User
  /// Sends a new verification code. Succeeds whether or not `email` has an
  /// account (doc 08).
  func resendVerificationCode(email: String) async throws(ServiceError)
  /// Signs in. The returned User may still be unverified.
  func signIn(email: String, password: String) async throws(ServiceError) -> User
  /// The User of the saved Session (FR-U1), or `nil` when there is none or
  /// the server no longer accepts it, in which case it is forgotten.
  func currentUser() async throws(ServiceError) -> User?
  /// Forgets the saved Session on this device.
  func forgetSession() async
}
