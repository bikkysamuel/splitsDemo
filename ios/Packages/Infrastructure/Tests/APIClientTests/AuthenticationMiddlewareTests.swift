import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import APIClient

/// ADR-0011, FR-U2: the access token goes on every operation the contract
/// marks bearerAuth, never on the anonymous ones; a 401 refreshes once and
/// retries once.
struct AuthenticationMiddlewareTests {
  @Test func addsTheBearerTokenToProtectedOperations() async throws {
    let tokens = FakeTokens(access: "token-1")
    let server = FakeServer(statuses: [.ok])

    _ = try await send(through: AuthenticationMiddleware(tokens: tokens), operationID: "getMe", to: server)

    #expect(await server.authorizations == ["Bearer token-1"])
  }

  @Test(arguments: [
    "signUp", "verifyEmail", "resendVerificationCode", "signIn", "refreshSession", "getHealthz", "getReadyz",
  ])
  func leavesAnonymousOperationsAlone(operationID: String) async throws {
    let tokens = FakeTokens(access: "token-1")
    let server = FakeServer(statuses: [.unauthorized])

    let response = try await send(
      through: AuthenticationMiddleware(tokens: tokens), operationID: operationID, to: server)

    // FR-U2: a 401 from sign-in or verification is a form error: no refresh.
    #expect(response.status == .unauthorized)
    #expect(await server.authorizations == [nil])
    #expect(await tokens.refreshCalls == 0)
  }

  @Test func sendsNoHeaderWithoutASavedToken() async throws {
    let tokens = FakeTokens(access: nil)
    let server = FakeServer(statuses: [.unauthorized])

    let response = try await send(through: AuthenticationMiddleware(tokens: tokens), operationID: "getMe", to: server)

    #expect(response.status == .unauthorized)
    #expect(await server.authorizations == [nil])
    #expect(await tokens.refreshCalls == 0)
  }

  @Test func refreshesOnceOn401AndRetriesWithTheNewToken() async throws {
    let tokens = FakeTokens(access: "old", refreshed: "new")
    let server = FakeServer(statuses: [.unauthorized, .ok])

    let response = try await send(
      through: AuthenticationMiddleware(tokens: tokens), operationID: "getMe", to: server, body: "{\"a\":1}")

    #expect(response.status == .ok)
    #expect(await server.authorizations == ["Bearer old", "Bearer new"])
    #expect(await tokens.rejected == ["old"])
    // The body is sent again on the retry.
    #expect(await server.bodies == ["{\"a\":1}", "{\"a\":1}"])
  }

  @Test func returnsThe401WhenTheRefreshEndsTheSession() async throws {
    let tokens = FakeTokens(access: "old", refreshed: nil)
    let server = FakeServer(statuses: [.unauthorized])

    let response = try await send(through: AuthenticationMiddleware(tokens: tokens), operationID: "getMe", to: server)

    #expect(response.status == .unauthorized)
    #expect(await server.authorizations == ["Bearer old"])
  }

  @Test func retriesOnlyOnce() async throws {
    let tokens = FakeTokens(access: "old", refreshed: "new")
    let server = FakeServer(statuses: [.unauthorized, .unauthorized])

    let response = try await send(through: AuthenticationMiddleware(tokens: tokens), operationID: "getMe", to: server)

    #expect(response.status == .unauthorized)
    #expect(await tokens.refreshCalls == 1)
  }

  @Test func aFailedRefreshThrows() async throws {
    let tokens = FakeTokens(access: "old", refreshError: URLError(.notConnectedToInternet))
    let server = FakeServer(statuses: [.unauthorized])

    await #expect(throws: URLError.self) {
      try await send(through: AuthenticationMiddleware(tokens: tokens), operationID: "getMe", to: server)
    }
  }

  @discardableResult
  private func send(
    through middleware: AuthenticationMiddleware, operationID: String, to server: FakeServer, body: String? = nil
  ) async throws -> HTTPResponse {
    let (response, _) = try await middleware.intercept(
      HTTPRequest(method: .post, scheme: "http", authority: "localhost", path: "/v1/x"),
      body: body.map { HTTPBody($0) },
      baseURL: try #require(URL(string: "http://localhost:8080")),
      operationID: operationID
    ) { request, body, _ in
      try await server.answer(request, body)
    }
    return response
  }
}

private actor FakeTokens: AccessTokenProvider {
  private var access: String?
  private let refreshed: String?
  private let refreshError: (any Error)?
  private(set) var refreshCalls = 0
  private(set) var rejected: [String] = []

  init(access: String?, refreshed: String? = nil, refreshError: (any Error)? = nil) {
    self.access = access
    self.refreshed = refreshed
    self.refreshError = refreshError
  }

  func accessToken() -> String? { access }

  func refreshAccessToken(rejected token: String) throws -> String? {
    refreshCalls += 1
    rejected.append(token)
    if let refreshError { throw refreshError }
    access = refreshed
    return refreshed
  }
}

private actor FakeServer {
  private var statuses: [HTTPResponse.Status]
  private(set) var authorizations: [String?] = []
  private(set) var bodies: [String] = []

  init(statuses: [HTTPResponse.Status]) {
    self.statuses = statuses
  }

  func answer(_ request: HTTPRequest, _ body: HTTPBody?) async throws -> (HTTPResponse, HTTPBody?) {
    authorizations.append(request.headerFields[.authorization])
    if let body {
      bodies.append(try await String(collecting: body, upTo: 1 << 16))
    }
    let status = statuses.isEmpty ? .ok : statuses.removeFirst()
    return (HTTPResponse(status: status), nil)
  }
}

/// The Go server writes RFC 3339 timestamps with fractional seconds, and
/// drops the fraction when it is zero; both must decode.
struct DateTranscoderTests {
  @Test(arguments: [
    ("2026-10-01T09:15:00Z", 1_790_846_100.0),
    ("2026-10-01T09:15:00.5Z", 1_790_846_100.5),
    ("2026-10-01T09:15:00.123456789Z", 1_790_846_100.123),
    ("2026-10-01T14:45:00+05:30", 1_790_846_100.0),
  ])
  func decodesRFC3339WithOrWithoutFractionalSeconds(text: String, seconds: Double) throws {
    let date = try RFC3339DateTranscoder().decode(text)

    #expect(abs(date.timeIntervalSince1970 - seconds) < 0.001)
  }

  @Test func refusesTextThatIsNotADate() {
    #expect(throws: (any Error).self) {
      try RFC3339DateTranscoder().decode("yesterday")
    }
  }

  @Test func encodesWithFractionalSeconds() throws {
    let text = try RFC3339DateTranscoder().encode(Date(timeIntervalSince1970: 1_790_846_100.5))

    #expect(text == "2026-10-01T09:15:00.500Z")
  }
}
