/// Why a server request failed, in Domain terms. The server's problem+json
/// `type` slugs map to `ProblemKind` (doc 07); the app maps each case to a
/// localized message.
public enum ServiceError: Error, Equatable, Sendable {
  /// One or more fields were refused (`validation-failed`).
  case invalidFields([FieldIssue])
  /// The server refused the request for a documented reason.
  case problem(ProblemKind)
  /// The request would cause something the User must confirm
  /// (`confirmation-required`); resend acknowledging these warning codes.
  case needsConfirmation([String])
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
  case adminRequired = "admin-required"
  case notFound = "not-found"
  case versionConflict = "version-conflict"
  case groupLimitReached = "group-limit-reached"
  case invalidCursor = "invalid-cursor"
  case memberLimitReached = "member-limit-reached"
  case memberNotEligible = "member-not-eligible"
  case groupClosed = "group-closed"
  case notCreator = "not-creator"
  case invalidState = "invalid-state"
  case confirmationRequired = "confirmation-required"
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
    case taken
    case notPositive
    case notGroupCurrency
    case notAMember
    case duplicateMember
    case noMembers
    case sameMember
    /// Exact amounts that don't sum to the total (on `split`).
    case exactSumMismatch
    /// Percentages that don't sum to exactly 100 (on `split`).
    case percentagesNot100
    /// A Member's entry the method can't use (on `split/members/N/input`).
    case missingInput, unexpectedInput, inputNotPositive, notWholeMinorUnits, tooManyDecimals, ratioNotInteger
    case other(String)

    public init(code: String) {
      switch code {
      case "required": self = .required
      case "invalid": self = .invalid
      case "too_short": self = .tooShort
      case "too_long": self = .tooLong
      case "too_common": self = .tooCommon
      case "taken": self = .taken
      case "not_positive": self = .notPositive
      case "not_group_currency": self = .notGroupCurrency
      case "not_a_member": self = .notAMember
      case "duplicate_member": self = .duplicateMember
      case "no_members": self = .noMembers
      case "same_member": self = .sameMember
      case "exact_sum_mismatch": self = .exactSumMismatch
      case "percentages_not_100": self = .percentagesNot100
      case "missing_input": self = .missingInput
      case "unexpected_input": self = .unexpectedInput
      case "input_not_positive": self = .inputNotPositive
      case "not_whole_minor_units": self = .notWholeMinorUnits
      case "too_many_decimals": self = .tooManyDecimals
      case "ratio_not_integer": self = .ratioNotInteger
      default: self = .other(code)
      }
    }
  }
}
