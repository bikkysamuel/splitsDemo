import APIClient
import Foundation
import OpenAPIRuntime

/// Supplies the saved access token and refreshes it on a 401 (ADR-0011,
/// FR-U2). Simultaneous 401s share one refresh. When the server refuses the
/// refresh token (expired, revoked or reused) the tokens are cleared and
/// `expirations` yields once; when the server can't be reached the tokens
/// are kept and the error is rethrown.
actor SessionRefresher: AccessTokenProvider {
  nonisolated let expirations: AsyncStream<Void>
  private let expirationsContinuation: AsyncStream<Void>.Continuation
  private let tokens: any TokenStore
  /// Calls `POST /v1/auth/refresh`. Anonymous, so it bypasses the auth
  /// middleware and can't recurse.
  private let client: Client
  private var inFlight: Task<String?, any Error>?

  init(serverURL: URL, transport: any ClientTransport, tokens: any TokenStore) {
    (expirations, expirationsContinuation) = AsyncStream.makeStream(of: Void.self)
    self.tokens = tokens
    self.client = Client(
      serverURL: serverURL, configuration: APIClientFactory.clientConfiguration, transport: transport)
  }

  func accessToken() async -> String? {
    await tokens.load()?.accessToken
  }

  func refreshAccessToken(rejected: String) async throws -> String? {
    if let inFlight {
      return try await inFlight.value
    }
    // Set before any suspension, so a second caller always finds it.
    let task = Task { try await self.refreshUnlessDone(rejected: rejected) }
    inFlight = task
    defer { inFlight = nil }
    return try await task.value
  }

  private func refreshUnlessDone(rejected: String) async throws -> String? {
    guard let saved = await tokens.load() else { return nil }
    // Another refresh finished after this request was sent.
    if saved.accessToken != rejected { return saved.accessToken }

    let output = try await client.refreshSession(body: .json(.init(refreshToken: saved.refreshToken)))
    switch output {
    case .ok(let ok):
      let session = try ok.body.json
      try await tokens.save(StoredTokens(session))
      return session.accessToken
    case .unauthorized, .badRequest:
      // 400: the saved refresh token isn't one the server can read, so it
      // can never work either.
      await tokens.clear()
      expirationsContinuation.yield()
      return nil
    case .contentTooLarge: throw RefreshUnavailable(status: 413)
    case .internalServerError: throw RefreshUnavailable(status: 500)
    case .undocumented(let status, _): throw RefreshUnavailable(status: status)
    }
  }
}

/// The refresh endpoint failed without refusing the token; the Session is
/// kept for a later try.
struct RefreshUnavailable: Error {
  let status: Int
}
