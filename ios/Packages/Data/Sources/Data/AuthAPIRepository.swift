import APIClient
import Domain
import Foundation
import OpenAPIRuntime

/// Sign-up, verification, sign-in and sign-out over the generated client.
/// Every new Session's tokens go to the `TokenStore` (the Keychain in the
/// app); the client's middleware attaches the access token to protected
/// requests and refreshes it on a 401 through `SessionRefresher`. Generated
/// types never leave this layer (ADR-0015).
public struct AuthAPIRepository: AuthRepository {
  private let client: Client
  private let tokens: any TokenStore
  private let refresher: SessionRefresher

  public init(configuration: APIConfiguration, tokens: any TokenStore) {
    self.init(serverURL: configuration.baseURL, transport: APIClientFactory.makeTransport(), tokens: tokens)
  }

  /// For tests: the same clients over another transport.
  init(serverURL: URL, transport: any ClientTransport, tokens: any TokenStore) {
    let refresher = SessionRefresher(serverURL: serverURL, transport: transport, tokens: tokens)
    self.client = Client(
      serverURL: serverURL,
      configuration: APIClientFactory.clientConfiguration,
      transport: transport,
      middlewares: [AuthenticationMiddleware(tokens: refresher)])
    self.tokens = tokens
    self.refresher = refresher
  }

  public var sessionExpirations: AsyncStream<Void> { refresher.expirations }

  public func signUp(email: String, password: String) async throws(ServiceError) -> User {
    let output = try await send { try await client.signUp(body: .json(.init(email: email, password: password))) }
    switch output {
    case .created(let created): return try await start(decoding { try created.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .conflict(let r): throw problem(409) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func verifyEmail(email: String, code: String) async throws(ServiceError) -> User {
    let output = try await send { try await client.verifyEmail(body: .json(.init(email: email, code: code))) }
    switch output {
    case .ok(let ok): return try await start(decoding { try ok.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func resendVerificationCode(email: String) async throws(ServiceError) {
    let output = try await send { try await client.resendVerificationCode(body: .json(.init(email: email))) }
    switch output {
    case .accepted: return
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func signIn(email: String, password: String) async throws(ServiceError) -> User {
    let output = try await send { try await client.signIn(body: .json(.init(email: email, password: password))) }
    switch output {
    case .ok(let ok): return try await start(decoding { try ok.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .tooManyRequests(let r): throw problem(429) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func currentUser() async throws(ServiceError) -> User? {
    guard await tokens.load() != nil else { return nil }
    let output = try await send { try await client.getMe() }
    switch output {
    case .ok(let ok):
      return try UserMapper.user(decoding { try ok.body.json })
    case .unauthorized:
      // The middleware already tried a refresh, and it failed.
      await tokens.clear()
      return nil
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func signOut() async {
    // Best effort: the Session is forgotten on this device whatever the
    // server answers; if it can't be reached, the Session expires on its own.
    _ = try? await client.signOut(headers: .init(idempotencyKey: UUID().uuidString))
    await tokens.clear()
  }

  /// Saves a new Session's tokens and returns its User.
  private func start(_ session: Components.Schemas.AuthSession) async throws(ServiceError) -> User {
    let user = try UserMapper.user(session.user)
    do {
      try await tokens.save(
        StoredTokens(
          accessToken: session.accessToken, accessExpiresAt: session.accessExpiresAt,
          refreshToken: session.refreshToken, refreshExpiresAt: session.refreshExpiresAt))
    } catch {
      throw .unexpected(status: nil)
    }
    return user
  }

  /// Runs a client call, mapping transport and decoding failures.
  private func send<Output>(_ call: () async throws -> Output) async throws(ServiceError) -> Output {
    do {
      return try await call()
    } catch {
      throw ServiceErrorMapper.transportError(error)
    }
  }

  /// Reads a response body; a body that doesn't match the contract is a
  /// server bug.
  private func decoding<Body>(_ read: () throws -> Body) throws(ServiceError) -> Body {
    do {
      return try read()
    } catch {
      throw .unexpected(status: nil)
    }
  }

  private func problem(_ status: Int, _ read: () throws -> Components.Schemas.Problem) -> ServiceError {
    guard let p = try? read() else { return .unexpected(status: status) }
    return ServiceErrorMapper.error(problem: p, status: status)
  }
}
