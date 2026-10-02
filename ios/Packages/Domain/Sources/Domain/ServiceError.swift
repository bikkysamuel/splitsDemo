/// Why a server request failed, in Domain terms. The server's problem+json
/// `type` slugs map to `ProblemKind` (doc 07); the app maps each case to a
/// localized message.
public enum ServiceError: Error, Equatable, Sendable {
  /// One or more fields were refused (`validation-failed`).
  case invalidFields([FieldIssue])
  /// The server refused the request for a documented reason.
  case problem(ProblemKind)
  /// The server could not be reached: no network, timeout, connection lost.
  case unreachable
  /// The server answered something the app doesn't understand, or failed;
  /// `status` is nil when there was no usable HTTP response.
  case unexpected(status: Int?)
}

/// The problem+json types the app knows (api/openapi.yaml, Problem schema).
/// An unknown type is reported as `ServiceError.unexpected`.
public enum ProblemKind: String, CaseIterable, Sendable {
  case invalidRequest = "invalid-request"
  case invalidCode = "invalid-code"
  case idempotencyKeyRequired = "idempotency-key-required"
  case unauthenticated
  case invalidCredentials = "invalid-credentials"
  case emailNotVerified = "email-not-verified"
  case emailTaken = "email-taken"
  case idempotencyKeyInProgress = "idempotency-key-in-progress"
  case requestTooLarge = "request-too-large"
  case idempotencyKeyReused = "idempotency-key-reused"
  case tooManyAttempts = "too-many-attempts"
  case notReady = "not-ready"
  case `internal`
}

/// One refused field of a request.
public struct FieldIssue: Equatable, Hashable, Sendable {
  /// The request field, such as `email` (the JSON Pointer without its `/`).
  public let field: String
  public let reason: Reason

  public init(field: String, reason: Reason) {
    self.field = field
    self.reason = reason
  }

  /// The server's field error codes (doc 07). Unknown codes keep their text.
  public enum Reason: Equatable, Hashable, Sendable {
    case required
    case invalid
    case tooShort
    case tooLong
    case tooCommon
    case other(String)

    public init(code: String) {
      switch code {
      case "required": self = .required
      case "invalid": self = .invalid
      case "too_short": self = .tooShort
      case "too_long": self = .tooLong
      case "too_common": self = .tooCommon
      default: self = .other(code)
      }
    }
  }
}
