import Domain
import Observation

/// Sign-in (FR-A4). Wrong credentials are a form error, never a session
/// expiry (FR-U2).
@MainActor
@Observable
public final class SignInViewModel {
  public var email = ""
  public var password = ""
  public private(set) var isSubmitting = false
  private(set) var errors = FormErrors()

  private let session: AppSession

  public init(session: AppSession) {
    self.session = session
  }

  public var canSubmit: Bool { !email.isEmpty && !password.isEmpty && !isSubmitting }

  public func submit() async {
    guard canSubmit else { return }
    isSubmitting = true
    defer { isSubmitting = false }
    do {
      let user = try await session.repository.signIn(email: email, password: password)
      errors = FormErrors()
      session.didAuthenticate(user)
    } catch {
      errors = FormErrors(error)
    }
  }
}
