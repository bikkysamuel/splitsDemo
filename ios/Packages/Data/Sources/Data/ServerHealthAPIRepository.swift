import APIClient
import Domain

/// Checks the server's liveness endpoint, `GET /healthz`, through the
/// generated client.
public struct ServerHealthAPIRepository: ServerHealthRepository {
  private let client: Client

  public init(client: Client) {
    self.client = client
  }

  public func checkHealth() async throws {
    switch try await client.getHealthz() {
    case .ok:
      return
    case .undocumented(let statusCode, _):
      throw ServerHealthError.unexpectedStatus(statusCode)
    }
  }
}
