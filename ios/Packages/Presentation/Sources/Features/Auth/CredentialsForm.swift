import Domain

/// What a form shows after a failed submit: one message for the screen and
/// one under each refused field, keyed by the request field name.
struct FormErrors: Equatable {
  var message: String?
  private(set) var fields: [String: String] = [:]

  init() {}

  init(_ error: ServiceError) {
    message = ServiceErrorMessage.key(for: error)
    guard case .invalidFields(let issues) = error else { return }
    for issue in issues where fields[issue.field] == nil {
      fields[issue.field] = ServiceErrorMessage.key(for: issue)
    }
  }

  var email: String? { fields["email"] }
  var password: String? { fields["password"] }
  subscript(field: String) -> String? { fields[field] }
}
