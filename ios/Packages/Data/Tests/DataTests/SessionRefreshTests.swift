import APIClient
import Domain
import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import Data

/// FR-U2, ADR-0011: a 401 refreshes once and retries; simultaneous 401s
/// share one refresh; a refused refresh ends the Session once.
struct SessionRefreshTests {
  @Test func a401RefreshesSavesTheNewPairAndRetries() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(me: .unauthorizedUnless("Bearer fixture-access-token"), refresh: .fixture("signin-200"))
    let repository = try makeRepository(server, tokens)

    let user = try await repository.currentUser()

    #expect(user?.email == "alice@example.com")
    #expect(await tokens.load()?.accessToken == "fixture-access-token")
    #expect(await tokens.load()?.refreshToken == "fixture-refresh-token")
    #expect(
      await server.log == [
        "GET /v1/me Bearer saved-access", "POST /v1/auth/refresh saved-refresh",
        "GET /v1/me Bearer fixture-access-token",
      ])
  }

  @Test func simultaneous401sShareOneRefresh() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(
      me: .unauthorizedUnless("Bearer fixture-access-token"), refresh: .fixture("signin-200"),
      refreshDelay: .milliseconds(50))
    let repository = try makeRepository(server, tokens)

    try await withThrowingTaskGroup(of: User?.self) { group in
      for _ in 0..<5 { group.addTask { try await repository.currentUser() } }
      for try await user in group { #expect(user?.email == "alice@example.com") }
    }

    #expect(await server.refreshCount == 1)
  }

  @Test func aRefusedRefreshEndsTheSessionOnce() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(
      me: .unauthorizedUnless("never"), refresh: .unauthorized, refreshDelay: .milliseconds(50))
    let repository = try makeRepository(server, tokens)
    let expirations = ExpirationCounter(repository.sessionExpirations)

    try await withThrowingTaskGroup(of: User?.self) { group in
      for _ in 0..<5 { group.addTask { try await repository.currentUser() } }
      for try await user in group { #expect(user == nil) }
    }

    #expect(await tokens.load() == nil)
    #expect(await server.refreshCount == 1)
    #expect(await expirations.count(settling: .milliseconds(100)) == 1)
  }

  // FR-U1: no connection must never throw a valid Session away.
  @Test func anUnreachableRefreshKeepsTheSession() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(me: .unauthorizedUnless("never"), refresh: .fail(URLError(.notConnectedToInternet)))
    let repository = try makeRepository(server, tokens)

    await #expect(throws: ServiceError.unreachable) {
      try await repository.currentUser()
    }
    #expect(await tokens.load() == .sample)
  }

  @Test func aServerErrorOnRefreshKeepsTheSession() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(me: .unauthorizedUnless("never"), refresh: .problem500)
    let repository = try makeRepository(server, tokens)

    await #expect(throws: ServiceError.unexpected(status: 500)) {
      try await repository.currentUser()
    }
    #expect(await tokens.load() == .sample)
  }

  @Test func signOutRevokesOnTheServerAndForgetsTheTokens() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(me: .unauthorizedUnless("never"), refresh: .unauthorized, signOut: .status(.noContent))
    let repository = try makeRepository(server, tokens)

    await repository.signOut()

    #expect(await tokens.load() == nil)
    let sent = try #require(await server.signOuts.first)
    #expect(sent.headerFields[.authorization] == "Bearer saved-access")
    let key = try #require(sent.headerFields[HTTPField.Name("Idempotency-Key")!])
    #expect(UUID(uuidString: key) != nil)
  }

  // A mid-session 401 on a write: refresh, then retry with the same
  // Idempotency-Key, so the server can replay instead of acting twice.
  @Test func aRetriedWriteKeepsItsIdempotencyKey() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(
      me: .unauthorizedUnless("never"), refresh: .fixture("signin-200"),
      signOut: .unauthorizedUnless("Bearer fixture-access-token"))
    let repository = try makeRepository(server, tokens)

    await repository.signOut()

    let sent = await server.signOuts
    #expect(sent.map { $0.headerFields[.authorization] } == ["Bearer saved-access", "Bearer fixture-access-token"])
    let keys = sent.map { $0.headerFields[HTTPField.Name("Idempotency-Key")!] }
    #expect(keys.count == 2 && keys[0] != nil && keys[0] == keys[1])
  }

  @Test func signOutForgetsTheTokensEvenWhenTheServerIsUnreachable() async throws {
    let tokens = InMemoryTokenStore(.sample)
    let server = RoutingServer(
      me: .unauthorizedUnless("never"), refresh: .unauthorized, signOut: .fail(URLError(.timedOut)))
    let repository = try makeRepository(server, tokens)

    await repository.signOut()

    #expect(await tokens.load() == nil)
  }

  private func makeRepository(_ server: RoutingServer, _ tokens: InMemoryTokenStore) throws -> AuthAPIRepository {
    AuthAPIRepository(serverURL: try #require(URL(string: "http://localhost:8080")), transport: server, tokens: tokens)
  }
}

/// Counts what an expiration stream yields.
private actor ExpirationCounter {
  private var seen = 0
  private var task: Task<Void, Never>?

  init(_ stream: AsyncStream<Void>) {
    Task { await self.start(stream) }
  }

  private func start(_ stream: AsyncStream<Void>) {
    task = Task {
      for await _ in stream { self.bump() }
    }
  }

  private func bump() { seen += 1 }

  /// Waits for late events, then returns the count.
  func count(settling delay: Duration) async -> Int {
    try? await Task.sleep(for: delay)
    task?.cancel()
    return seen
  }
}

/// Answers by route, like the server would.
private final class RoutingServer: ClientTransport, Sendable {
  enum Answer: Sendable {
    case fixture(String)
    case status(HTTPResponse.Status)
    case unauthorized
    case problem500
    case fail(any Error)
    /// 200 with the me fixture for this Authorization value, 401 otherwise.
    case unauthorizedUnless(String)
  }

  private let me: Answer
  private let refresh: Answer
  private let signOutAnswer: Answer
  private let refreshDelay: Duration
  private let state = State()

  init(me: Answer, refresh: Answer, signOut: Answer = .status(.noContent), refreshDelay: Duration = .zero) {
    self.me = me
    self.refresh = refresh
    self.signOutAnswer = signOut
    self.refreshDelay = refreshDelay
  }

  var log: [String] { get async { await state.log } }
  var refreshCount: Int { get async { await state.refreshCount } }
  var signOuts: [HTTPRequest] { get async { await state.signOuts } }

  func send(
    _ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String
  ) async throws -> (HTTPResponse, HTTPBody?) {
    let auth = request.headerFields[.authorization] ?? "-"
    switch operationID {
    case "getMe":
      await state.append("GET /v1/me \(auth)")
      return try answer(me, auth: auth)
    case "refreshSession":
      let data = try await Data(collecting: body ?? HTTPBody(), upTo: 1 << 16)
      let token = (try JSONSerialization.jsonObject(with: data) as? [String: String])?["refresh_token"] ?? "?"
      await state.refreshed("POST /v1/auth/refresh \(token)")
      try await Task.sleep(for: refreshDelay)
      return try answer(refresh, auth: auth)
    case "signOut":
      await state.signedOut(request)
      return try answer(signOutAnswer, auth: auth)
    default:
      return (HTTPResponse(status: .notFound), nil)
    }
  }

  private func answer(_ answer: Answer, auth: String) throws -> (HTTPResponse, HTTPBody?) {
    switch answer {
    case .fixture(let name):
      return try json(name, status: .ok)
    case .status(let status):
      return (HTTPResponse(status: status), nil)
    case .unauthorized:
      return try json("me-401-unauthenticated", status: .unauthorized, problem: true)
    case .problem500:
      var fields = HTTPFields()
      fields[.contentType] = "application/problem+json"
      return (
        HTTPResponse(status: .internalServerError, headerFields: fields),
        HTTPBody(#"{"type":"https://splits.dev/problems/internal","title":"Internal error","status":500}"#)
      )
    case .fail(let error):
      throw error
    case .unauthorizedUnless(let accepted):
      return auth == accepted
        ? try json("me-200", status: .ok) : try json("me-401-unauthenticated", status: .unauthorized, problem: true)
    }
  }

  private func json(
    _ name: String, status: HTTPResponse.Status, problem: Bool = false
  ) throws -> (
    HTTPResponse, HTTPBody?
  ) {
    let url = try #require(Bundle.module.url(forResource: name, withExtension: "json", subdirectory: "Fixtures"))
    var fields = HTTPFields()
    fields[.contentType] = problem ? "application/problem+json" : "application/json"
    return (HTTPResponse(status: status, headerFields: fields), HTTPBody(try Data(contentsOf: url)))
  }

  private actor State {
    var log: [String] = []
    var refreshCount = 0
    var signOuts: [HTTPRequest] = []
    func append(_ line: String) { log.append(line) }
    func refreshed(_ line: String) {
      log.append(line)
      refreshCount += 1
    }
    func signedOut(_ request: HTTPRequest) { signOuts.append(request) }
  }
}
