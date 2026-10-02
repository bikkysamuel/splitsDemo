import Domain
import Observation

/// Sign-up (FR-A1, FR-A3). The server checks the email and password; this
/// screen shows what it refused.
@MainActor
@Observable
public final class SignUpViewModel {
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
      let user = try await session.repository.signUp(email: email, password: password)
      errors = FormErrors()
      session.didAuthenticate(user)
    } catch {
      errors = FormErrors(error)
    }
  }
}
