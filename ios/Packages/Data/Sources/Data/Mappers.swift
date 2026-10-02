import APIClient
import Domain
import Foundation
import OpenAPIRuntime

/// Generated `User` → Domain `User`.
enum UserMapper {
  static func user(_ api: Components.Schemas.User) throws(ServiceError) -> User {
    guard let id = UUID(uuidString: api.id) else { throw .unexpected(status: nil) }
    return User(id: id, email: api.email, emailVerified: api.emailVerified)
  }
}

/// problem+json and transport failures → `ServiceError` (doc 07).
enum ServiceErrorMapper {
  static let problemTypeBase = "https://splits.dev/problems/"
  private static let validationFailed = "validation-failed"

  static func error(problem: Components.Schemas.Problem, status: Int) -> ServiceError {
    guard problem._type.hasPrefix(problemTypeBase) else { return .unexpected(status: status) }
    let slug = String(problem._type.dropFirst(problemTypeBase.count))
    if slug == validationFailed {
      let issues = (problem.errors ?? []).map { error in
        FieldIssue(
          field: error.field.hasPrefix("/") ? String(error.field.dropFirst()) : error.field,
          reason: FieldIssue.Reason(code: error.code))
      }
      return .invalidFields(issues)
    }
    guard let kind = ProblemKind(rawValue: slug) else { return .unexpected(status: status) }
    return .problem(kind)
  }

  /// A failure before any documented response: no network, a timeout, or a
  /// body the client could not decode.
  static func transportError(_ error: any Error) -> ServiceError {
    var current: any Error = error
    while true {
      if current is URLError { return .unreachable }
      if let client = current as? ClientError {
        current = client.underlyingError
        continue
      }
      return .unexpected(status: nil)
    }
  }
}
