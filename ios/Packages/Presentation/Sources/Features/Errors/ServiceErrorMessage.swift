import Domain

/// String Catalog keys for what went wrong (doc 05: problem `type` → typed
/// Domain error → localized message). Field issues are shown under their
/// field; everything else as one message on the screen.
enum ServiceErrorMessage {
  static let generic = "Something went wrong. Try again."
  static let unreachable = "Can't reach the server. Check your connection and try again."
  static let checkFields = "Check the fields marked below."

  /// The screen-level message for an error.
  static func key(for error: ServiceError) -> String {
    switch error {
    case .invalidFields: checkFields
    case .problem(let kind): key(for: kind)
    case .unreachable: unreachable
    case .unexpected: generic
    }
  }

  static func key(for kind: ProblemKind) -> String {
    switch kind {
    case .invalidCredentials: "The email or password is wrong."
    case .emailTaken: "An account with this email already exists. Sign in instead."
    case .invalidCode: "That code didn't work. Check it, or send a new one."
    case .unauthenticated: "Your session has ended. Sign in again."
    case .emailNotVerified: "Verify your email first."
    case .requestTooLarge: "That's too much text to send."
    case .invalidRequest, .idempotencyKeyRequired, .idempotencyKeyInProgress, .idempotencyKeyReused, .notReady,
      .internal:
      generic
    }
  }

  /// The message under a refused field.
  static func key(for issue: FieldIssue) -> String {
    switch (issue.field, issue.reason) {
    case ("email", .required): "Enter your email address."
    case ("email", _): "Enter a valid email address."
    case ("password", .required): "Enter a password."
    case ("password", .tooShort): "Use at least 10 characters."
    case ("password", .tooLong): "Use at most 128 characters."
    case ("password", .tooCommon): "This password is too common. Choose another."
    default: "This field is invalid."
    }
  }

  /// Every key the catalog must hold, for StringCatalogTests.
  static var allKeys: [String] {
    var keys = [generic, unreachable, checkFields] + ProblemKind.allCases.map(key(for:))
    let reasons: [FieldIssue.Reason] = [.required, .invalid, .tooShort, .tooLong, .tooCommon, .other("x")]
    for field in ["email", "password", "other"] {
      keys += reasons.map { key(for: FieldIssue(field: field, reason: $0)) }
    }
    return Array(Set(keys)).sorted()
  }
}
