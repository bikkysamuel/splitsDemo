import APIClient
import Domain
import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import Data

/// The Data seam (doc 09): the repository runs the generated client over a
/// fake transport answering with JSON recorded from the real server
/// (Fixtures/, tokens and emails replaced with fake values).
struct AuthAPIRepositoryTests {
  // MARK: Successful sign-up, verification and sign-in

  @Test func signUpReturnsTheUnverifiedUserAndSavesTheTokens() async throws {
    let (repository, tokens, _) = try makeRepository(.fixture("signup-201", status: .created))

    let user = try await repository.signUp(email: "alice@example.com", password: "correct horse battery")

    #expect(
      user
        == User(
          id: try #require(UUID(uuidString: "01a0f963-a5df-76b2-9991-df0c21402fd6")), email: "alice@example.com",
          emailVerified: false))
    let saved = try #require(await tokens.load())
    #expect(saved.accessToken == "fixture-access-token")
    #expect(saved.refreshToken == "fixture-refresh-token")
    // Go writes nanoseconds; the date must decode, to the millisecond.
    let expected = try RFC3339DateTranscoder().decode("2026-10-01T21:49:12.447Z")
    #expect(abs(saved.accessExpiresAt.timeIntervalSince(expected)) < 0.001)
  }

  @Test func signUpSendsTheEmailAndPasswordAsJSON() async throws {
    let (repository, _, transport) = try makeRepository(.fixture("signup-201", status: .created))

    _ = try await repository.signUp(email: "alice@example.com", password: "correct horse battery")

    let sent = try #require(await transport.requests.first)
    #expect(sent.request.method == .post)
    #expect(sent.request.path == "/v1/auth/signup")
    let body = try JSONSerialization.jsonObject(with: sent.body) as? [String: String]
    #expect(body == ["email": "alice@example.com", "password": "correct horse battery"])
    #expect(sent.request.headerFields[.authorization] == nil)
  }

  @Test func verifyEmailReturnsTheVerifiedUserAndReplacesTheTokens() async throws {
    let tokens = InMemoryTokenStore(
      .init(accessToken: "old", accessExpiresAt: .now, refreshToken: "old", refreshExpiresAt: .now))
    let (repository, _, _) = try makeRepository(.fixture("verify-200"), tokens: tokens)

    let user = try await repository.verifyEmail(email: "alice@example.com", code: "123456")

    #expect(user.emailVerified)
    #expect(await tokens.load()?.accessToken == "fixture-access-token")
  }

  @Test func signInReturnsTheUserAndSavesTheTokens() async throws {
    let (repository, tokens, _) = try makeRepository(.fixture("signin-200"))

    let user = try await repository.signIn(email: "alice@example.com", password: "correct horse battery")

    #expect(user.emailVerified)
    #expect(await tokens.load()?.refreshToken == "fixture-refresh-token")
  }

  @Test func resendSucceedsOn202() async throws {
    let (repository, _, transport) = try makeRepository(.empty(status: .accepted))

    try await repository.resendVerificationCode(email: "alice@example.com")

    #expect(await transport.requests.first?.request.path == "/v1/auth/verify-email/resend")
  }

  // MARK: Problem types → typed Domain errors

  @Test func validationFailuresListEachField() async throws {
    let (repository, tokens, _) = try makeRepository(
      .fixture("signup-400-validation-failed", status: .badRequest, problem: true))

    await #expect(
      throws: ServiceError.invalidFields([
        FieldIssue(field: "email", reason: .invalid), FieldIssue(field: "password", reason: .tooShort),
      ])
    ) {
      try await repository.signUp(email: "nope", password: "short")
    }
    #expect(await tokens.load() == nil)
  }

  @Test func aTakenEmailIsEmailTaken() async throws {
    let (repository, _, _) = try makeRepository(.fixture("signup-409-email-taken", status: .conflict, problem: true))

    await #expect(throws: ServiceError.problem(.emailTaken)) {
      try await repository.signUp(email: "alice@example.com", password: "correct horse battery")
    }
  }

  @Test func aWrongCodeIsInvalidCode() async throws {
    let (repository, _, _) = try makeRepository(.fixture("verify-400-invalid-code", status: .badRequest, problem: true))

    await #expect(throws: ServiceError.problem(.invalidCode)) {
      try await repository.verifyEmail(email: "alice@example.com", code: "000000")
    }
  }

  @Test func wrongCredentialsAreInvalidCredentials() async throws {
    let (repository, tokens, _) = try makeRepository(
      .fixture("signin-401-invalid-credentials", status: .unauthorized, problem: true))

    await #expect(throws: ServiceError.problem(.invalidCredentials)) {
      try await repository.signIn(email: "alice@example.com", password: "wrong password!!")
    }
    #expect(await tokens.load() == nil)
  }

  @Test func anUnknownProblemTypeIsUnexpected() async throws {
    let (repository, _, _) = try makeRepository(
      .json(
        #"{"type":"https://splits.dev/problems/from-the-future","title":"New","status":400}"#, status: .badRequest,
        problem: true))

    await #expect(throws: ServiceError.unexpected(status: 400)) {
      try await repository.signUp(email: "alice@example.com", password: "correct horse battery")
    }
  }

  @Test func anUndocumentedStatusIsUnexpected() async throws {
    let (repository, _, _) = try makeRepository(.json("{}", status: .badGateway))

    await #expect(throws: ServiceError.unexpected(status: 502)) {
      try await repository.signIn(email: "alice@example.com", password: "correct horse battery")
    }
  }

  @Test func aMalformedSuccessBodyIsUnexpected() async throws {
    let (repository, tokens, _) = try makeRepository(.json(#"{"user":{}}"#))

    await #expect(throws: ServiceError.unexpected(status: nil)) {
      try await repository.signIn(email: "alice@example.com", password: "correct horse battery")
    }
    #expect(await tokens.load() == nil)
  }

  @Test(arguments: [URLError.Code.notConnectedToInternet, .timedOut, .cannotConnectToHost, .networkConnectionLost])
  func transportFailuresAreUnreachable(code: URLError.Code) async throws {
    let (repository, _, _) = try makeRepository(.fail(URLError(code)))

    await #expect(throws: ServiceError.unreachable) {
      try await repository.signIn(email: "alice@example.com", password: "correct horse battery")
    }
  }

  // MARK: Launch check (FR-U1)

  @Test func currentUserIsNilWithoutSavedTokensAndAsksNothing() async throws {
    let (repository, _, transport) = try makeRepository(.fixture("me-200"))

    let user = try await repository.currentUser()

    #expect(user == nil)
    #expect(await transport.requests.isEmpty)
  }

  @Test func currentUserSendsTheSavedAccessToken() async throws {
    let (repository, _, transport) = try makeRepository(.fixture("me-200"), tokens: InMemoryTokenStore(.sample))

    let user = try await repository.currentUser()

    #expect(user?.email == "alice@example.com")
    let sent = try #require(await transport.requests.first)
    #expect(sent.request.path == "/v1/me")
    #expect(sent.request.headerFields[.authorization] == "Bearer saved-access")
  }

  @Test func currentUserForgetsASessionTheServerRefuses() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let (repository, _, _) = try makeRepository(
      .fixture("me-401-unauthenticated", status: .unauthorized, problem: true), tokens: tokens)

    let user = try await repository.currentUser()

    #expect(user == nil)
    #expect(await tokens.load() == nil)
  }

  // FR-U1: no connection must never throw a valid Session away.
  @Test func currentUserKeepsTheSessionWhenTheServerCannotBeReached() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let (repository, _, _) = try makeRepository(.fail(URLError(.notConnectedToInternet)), tokens: tokens)

    await #expect(throws: ServiceError.unreachable) {
      try await repository.currentUser()
    }
    #expect(await tokens.load() == .sample)
  }

  @Test func forgetSessionClearsTheTokens() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let (repository, _, _) = try makeRepository(.empty(status: .ok), tokens: tokens)

    await repository.forgetSession()

    #expect(await tokens.load() == nil)
  }

  private func makeRepository(
    _ behaviour: RecordingTransport.Behaviour, tokens: InMemoryTokenStore = InMemoryTokenStore()
  ) throws -> (AuthAPIRepository, InMemoryTokenStore, RecordingTransport) {
    let transport = RecordingTransport(behaviour: behaviour)
    let repository = AuthAPIRepository(
      serverURL: try #require(URL(string: "http://localhost:8080")), transport: transport, tokens: tokens)
    return (repository, tokens, transport)
  }
}

extension StoredTokens {
  static let sample = StoredTokens(
    accessToken: "saved-access", accessExpiresAt: Date(timeIntervalSince1970: 1_790_846_100),
    refreshToken: "saved-refresh", refreshExpiresAt: Date(timeIntervalSince1970: 1_793_438_100))
}

/// Answers every request the same way and records what was sent.
final class RecordingTransport: ClientTransport, Sendable {
  enum Behaviour: Sendable {
    case answer(status: HTTPResponse.Status, body: Data, contentType: String)
    case fail(any Error)

    static func fixture(_ name: String, status: HTTPResponse.Status = .ok, problem: Bool = false) throws -> Behaviour {
      let url = try #require(Bundle.module.url(forResource: name, withExtension: "json", subdirectory: "Fixtures"))
      return .answer(status: status, body: try Data(contentsOf: url), contentType: contentType(problem))
    }

    static func json(_ text: String, status: HTTPResponse.Status = .ok, problem: Bool = false) -> Behaviour {
      .answer(status: status, body: Data(text.utf8), contentType: contentType(problem))
    }

    static func empty(status: HTTPResponse.Status) -> Behaviour {
      .answer(status: status, body: Data(), contentType: "")
    }

    private static func contentType(_ problem: Bool) -> String {
      problem ? "application/problem+json" : "application/json"
    }
  }

  struct Sent: Sendable {
    let request: HTTPRequest
    let body: Data
  }

  let behaviour: Behaviour
  private let recorder = Recorder()

  init(behaviour: Behaviour) {
    self.behaviour = behaviour
  }

  var requests: [Sent] {
    get async { await recorder.sent }
  }

  func send(
    _ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String
  ) async throws -> (HTTPResponse, HTTPBody?) {
    let data: Data
    if let body {
      data = try await Data(collecting: body, upTo: 1 << 20)
    } else {
      data = Data()
    }
    await recorder.record(Sent(request: request, body: data))
    switch behaviour {
    case .answer(let status, let body, let contentType):
      var fields = HTTPFields()
      if !contentType.isEmpty { fields[.contentType] = contentType }
      return (HTTPResponse(status: status, headerFields: fields), body.isEmpty ? nil : HTTPBody(body))
    case .fail(let error):
      throw error
    }
  }

  private actor Recorder {
    var sent: [Sent] = []
    func record(_ s: Sent) { sent.append(s) }
  }
}
