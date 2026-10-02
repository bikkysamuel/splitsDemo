import APIClient
import Domain

/// Checks the server's liveness endpoint, `GET /healthz`, through the
/// generated client. The generated `Client` stays inside Data (ADR-0015).
public struct ServerHealthAPIRepository: ServerHealthRepository {
  private let client: Client

  public init(configuration: APIConfiguration) {
    self.init(client: APIClientFactory.makeAnonymousClient(configuration: configuration))
  }

  init(client: Client) {
    self.client = client
  }

  public func checkHealth() async throws {
    switch try await client.getHealthz() {
    case .ok:
      return
    case .internalServerError:
      throw ServerHealthError.unexpectedStatus(500)
    case .undocumented(let statusCode, _):
      throw ServerHealthError.unexpectedStatus(statusCode)
    }
  }
}
