import Domain

/// What a sign-up or sign-in screen shows after a failed submit: one
/// message for the screen and one under each refused field.
struct FormErrors: Equatable {
  var message: String?
  var email: String?
  var password: String?

  init() {}

  init(_ error: ServiceError) {
    message = ServiceErrorMessage.key(for: error)
    guard case .invalidFields(let issues) = error else { return }
    for issue in issues {
      switch issue.field {
      case "email": email = email ?? ServiceErrorMessage.key(for: issue)
      case "password": password = password ?? ServiceErrorMessage.key(for: issue)
      default: break
      }
    }
  }
}
