import Foundation
import HTTPTypes
import OpenAPIRuntime

/// Supplies the Session's access token and refreshes it (ADR-0011). The
/// caller keeps the tokens in the Keychain; this module never stores them.
public protocol AccessTokenProvider: Sendable {
  /// The saved access token, or `nil` when there is no Session.
  func accessToken() async -> String?
  /// Called when a request sent with `rejected` answered 401. Returns the
  /// access token to retry with, or `nil` when the Session is over.
  /// Simultaneous callers must share one refresh. Throws when the refresh
  /// could not reach the server, so the Session is kept.
  func refreshAccessToken(rejected: String) async throws -> String?
}

/// Puts `Authorization: Bearer <access token>` on every operation except
/// the anonymous ones, and on a 401 refreshes once and retries once
/// (FR-U2). Anonymous operations, such as sign-in, pass through untouched,
/// so their 401s stay form errors.
public struct AuthenticationMiddleware: ClientMiddleware {
  /// Operations the contract leaves without bearerAuth (api/openapi.yaml).
  /// Every other operation gets the token, so a new one is covered by
  /// default.
  static let anonymousOperations: Set<String> = [
    "getHealthz", "getReadyz", "signUp", "verifyEmail", "resendVerificationCode", "signIn", "refreshSession",
  ]

  /// Request bodies are at most 64 KB (doc 08), so a body is buffered for
  /// the retry.
  static let maxBodyBytes = 64 << 10

  private let tokens: any AccessTokenProvider

  public init(tokens: any AccessTokenProvider) {
    self.tokens = tokens
  }

  public func intercept(
    _ request: HTTPRequest,
    body: HTTPBody?,
    baseURL: URL,
    operationID: String,
    next: @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
  ) async throws -> (HTTPResponse, HTTPBody?) {
    guard !Self.anonymousOperations.contains(operationID), let token = await tokens.accessToken() else {
      return try await next(request, body, baseURL)
    }
    let bytes: Data? =
      if let body { try await Data(collecting: body, upTo: Self.maxBodyBytes) } else { nil }
    var request = request
    request.headerFields[.authorization] = "Bearer \(token)"
    let first = try await next(request, bytes.map { HTTPBody($0) }, baseURL)
    guard first.0.status == .unauthorized, let fresh = try await tokens.refreshAccessToken(rejected: token) else {
      return first
    }
    request.headerFields[.authorization] = "Bearer \(fresh)"
    return try await next(request, bytes.map { HTTPBody($0) }, baseURL)
  }
}
