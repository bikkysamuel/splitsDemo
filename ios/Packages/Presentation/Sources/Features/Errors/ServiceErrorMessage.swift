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
    case .tooManyAttempts: "Too many attempts. Wait a moment, then try again."
    case .adminRequired: "Only an Admin of this Group can do this."
    case .notFound: "This isn't available to you."
    case .versionConflict: "Someone else changed this just now. Reload and try again."
    case .groupLimitReached: "You're already in 200 Groups, the most allowed."
    case .memberLimitReached: "This Group already has 50 Members, the most allowed."
    case .memberNotEligible: "Only a Member who has an account can be an Admin."
    case .groupClosed: "This Group is closed. Reopen it to make changes."
    case .invalidRequest, .idempotencyKeyRequired, .idempotencyKeyInProgress, .idempotencyKeyReused, .notReady,
      .internal, .invalidCursor:
      generic
    }
  }

  /// The message under a refused field.
  static func key(for issue: FieldIssue) -> String {
    switch (issue.field, issue.reason) {
    case ("email", .required): "Enter your email address."
    case ("email", .taken): "Someone in this Group already has this email."
    case ("email", _): "Enter a valid email address."
    case ("password", .required): "Enter a password."
    case ("password", .tooShort): "Use at least 10 characters."
    case ("password", .tooLong): "Use at most 128 characters."
    case ("password", .tooCommon): "This password is too common. Choose another."
    case ("name", .required): "Enter a name."
    case ("name", .tooLong): "Use at most 100 characters."
    case ("display_name", .required): "Enter a name."
    case ("display_name", .tooLong): "Use at most 50 characters."
    case ("display_name", .taken): "Someone in this Group already has this name."
    case ("currency", _): "Choose a currency."
    default: "This field is invalid."
    }
  }

  /// Every key the catalog must hold, for StringCatalogTests.
  static var allKeys: [String] {
    var keys = [generic, unreachable, checkFields] + ProblemKind.allCases.map(key(for:))
    let reasons: [FieldIssue.Reason] = [.required, .invalid, .tooShort, .tooLong, .tooCommon, .taken, .other("x")]
    for field in ["email", "password", "name", "display_name", "currency", "other"] {
      keys += reasons.map { key(for: FieldIssue(field: field, reason: $0)) }
    }
    return Array(Set(keys)).sorted()
  }
}
