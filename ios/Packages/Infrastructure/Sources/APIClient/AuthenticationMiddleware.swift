import Foundation
import HTTPTypes
import OpenAPIRuntime

/// Puts `Authorization: Bearer <access token>` on every operation except
/// the anonymous ones (ADR-0011). The token comes from the caller, which
/// keeps it in the Keychain; this module never stores it.
public struct AuthenticationMiddleware: ClientMiddleware {
  /// Operations the contract leaves without bearerAuth (api/openapi.yaml).
  /// Every other operation gets the token, so a new one is covered by
  /// default.
  static let anonymousOperations: Set<String> = [
    "getHealthz", "getReadyz", "signUp", "verifyEmail", "resendVerificationCode", "signIn",
  ]

  private let accessToken: @Sendable () async -> String?

  public init(accessToken: @escaping @Sendable () async -> String?) {
    self.accessToken = accessToken
  }

  public func intercept(
    _ request: HTTPRequest,
    body: HTTPBody?,
    baseURL: URL,
    operationID: String,
    next: @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
  ) async throws -> (HTTPResponse, HTTPBody?) {
    var request = request
    if !Self.anonymousOperations.contains(operationID), let token = await accessToken() {
      request.headerFields[.authorization] = "Bearer \(token)"
    }
    return try await next(request, body, baseURL)
  }
}
