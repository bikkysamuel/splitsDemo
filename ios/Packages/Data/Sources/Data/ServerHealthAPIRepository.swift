import APIClient
import Domain

/// Checks the server's liveness endpoint, `GET /healthz`.
public struct ServerHealthAPIRepository: ServerHealthRepository {
  private let client: HTTPClient

  public init(client: HTTPClient) {
    self.client = client
  }

  public func checkHealth() async throws {
    let status = try await client.getStatusCode("healthz")
    guard status == 200 else {
      throw ServerHealthError.unexpectedStatus(status)
    }
  }
}

/// Why the health check failed.
public enum ServerHealthError: Error, Equatable {
  case unexpectedStatus(Int)
}
