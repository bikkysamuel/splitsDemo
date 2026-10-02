import Domain
import Foundation

/// A scripted `AuthRepository` for ViewModel tests (the ADR-0015 seam).
actor FakeAuthRepository: AuthRepository {
  struct Call: Equatable {
    let name: String
    let email: String
    let secret: String
  }

  var signUpResult: Result<User, ServiceError> = .success(.unverified)
  var verifyResult: Result<User, ServiceError> = .success(.verified)
  var resendResult: Result<Void, ServiceError> = .success(())
  var signInResult: Result<User, ServiceError> = .success(.verified)
  var currentUserResult: Result<User?, ServiceError> = .success(nil)
  private(set) var calls: [Call] = []
  private(set) var forgotSession = false

  func set(signUp: Result<User, ServiceError>) { signUpResult = signUp }
  func set(verify: Result<User, ServiceError>) { verifyResult = verify }
  func set(resend: Result<Void, ServiceError>) { resendResult = resend }
  func set(signIn: Result<User, ServiceError>) { signInResult = signIn }
  func set(currentUser: Result<User?, ServiceError>) { currentUserResult = currentUser }

  func signUp(email: String, password: String) async throws(ServiceError) -> User {
    calls.append(Call(name: "signUp", email: email, secret: password))
    return try signUpResult.get()
  }

  func verifyEmail(email: String, code: String) async throws(ServiceError) -> User {
    calls.append(Call(name: "verifyEmail", email: email, secret: code))
    return try verifyResult.get()
  }

  func resendVerificationCode(email: String) async throws(ServiceError) {
    calls.append(Call(name: "resend", email: email, secret: ""))
    try resendResult.get()
  }

  func signIn(email: String, password: String) async throws(ServiceError) -> User {
    calls.append(Call(name: "signIn", email: email, secret: password))
    return try signInResult.get()
  }

  func currentUser() async throws(ServiceError) -> User? {
    calls.append(Call(name: "currentUser", email: "", secret: ""))
    return try currentUserResult.get()
  }

  func forgetSession() async {
    forgotSession = true
  }
}

extension User {
  static let unverified = User(
    id: UUID(), email: "alice@example.com", emailVerified: false)
  static let verified = User(id: unverified.id, email: unverified.email, emailVerified: true)
}
