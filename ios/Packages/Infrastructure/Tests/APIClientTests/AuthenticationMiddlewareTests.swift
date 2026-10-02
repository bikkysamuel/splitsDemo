import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import APIClient

/// ADR-0011: the access token goes on every operation the contract marks
/// bearerAuth, and never on the anonymous auth endpoints.
struct AuthenticationMiddlewareTests {
  @Test func addsTheBearerTokenToProtectedOperations() async throws {
    let middleware = AuthenticationMiddleware(accessToken: { "token-1" })

    let sent = try await send(through: middleware, operationID: "getMe")

    #expect(sent.headerFields[.authorization] == "Bearer token-1")
  }

  @Test(arguments: ["signUp", "verifyEmail", "resendVerificationCode", "signIn", "getHealthz", "getReadyz"])
  func leavesAnonymousOperationsAlone(operationID: String) async throws {
    let middleware = AuthenticationMiddleware(accessToken: { "token-1" })

    let sent = try await send(through: middleware, operationID: operationID)

    #expect(sent.headerFields[.authorization] == nil)
  }

  @Test func sendsNoHeaderWithoutASavedToken() async throws {
    let middleware = AuthenticationMiddleware(accessToken: { nil })

    let sent = try await send(through: middleware, operationID: "getMe")

    #expect(sent.headerFields[.authorization] == nil)
  }

  private func send(through middleware: AuthenticationMiddleware, operationID: String) async throws -> HTTPRequest {
    let recorder = RequestRecorder()
    _ = try await middleware.intercept(
      HTTPRequest(method: .get, scheme: "http", authority: "localhost", path: "/v1/me"),
      body: nil,
      baseURL: try #require(URL(string: "http://localhost:8080")),
      operationID: operationID
    ) { request, _, _ in
      await recorder.record(request)
      return (HTTPResponse(status: .ok), nil)
    }
    return try #require(await recorder.request)
  }
}

private actor RequestRecorder {
  var request: HTTPRequest?
  func record(_ request: HTTPRequest) { self.request = request }
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
