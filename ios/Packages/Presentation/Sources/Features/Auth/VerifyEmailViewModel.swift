import Domain
import Observation

/// Email verification with the 6-digit code (FR-A2; development: `123456`).
@MainActor
@Observable
public final class VerifyEmailViewModel {
  nonisolated public static let codeLength = 6

  public let email: String
  /// The code as typed; only digits are kept, at most six.
  public var code = "" {
    didSet {
      let digits = String(code.filter(\.isASCIIDigit).prefix(Self.codeLength))
      if digits != code { code = digits }
    }
  }
  public private(set) var isSubmitting = false
  public private(set) var isResending = false
  /// The screen's message key: an error, or the resend confirmation.
  private(set) var message: String?
  private(set) var messageIsError = false

  nonisolated static let resentMessage = "We sent a new code."

  private let session: AppSession

  public init(email: String, session: AppSession) {
    self.email = email
    self.session = session
  }

  public var canSubmit: Bool { code.count == Self.codeLength && !isSubmitting }

  public func submit() async {
    guard canSubmit else { return }
    isSubmitting = true
    defer { isSubmitting = false }
    do {
      let user = try await session.repository.verifyEmail(email: email, code: code)
      message = nil
      session.didAuthenticate(user)
    } catch {
      show(error)
    }
  }

  public func resend() async {
    guard !isResending else { return }
    isResending = true
    defer { isResending = false }
    do {
      try await session.repository.resendVerificationCode(email: email)
      code = ""
      message = Self.resentMessage
      messageIsError = false
    } catch {
      show(error)
    }
  }

  public func useAnotherAccount() async {
    await session.useAnotherAccount()
  }

  private func show(_ error: ServiceError) {
    message = ServiceErrorMessage.key(for: error)
    messageIsError = true
  }
}

extension Character {
  fileprivate var isASCIIDigit: Bool { isASCII && isNumber }
}
