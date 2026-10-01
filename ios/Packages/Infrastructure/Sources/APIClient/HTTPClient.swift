import Foundation

/// Sends requests to the API. A hand-written stand-in until #7 generates the
/// client from `api/openapi.yaml` (ADR-0012).
public struct HTTPClient: Sendable {
  private let configuration: APIConfiguration
  private let session: URLSession

  public init(configuration: APIConfiguration, session: URLSession = URLSessionFactory.makeSession()) {
    self.configuration = configuration
    self.session = session
  }

  /// GETs `path` (relative to the base URL) and returns the HTTP status code.
  public func getStatusCode(_ path: String) async throws -> Int {
    let url = configuration.baseURL.appending(path: path)
    let (_, response) = try await session.data(from: url)
    guard let http = response as? HTTPURLResponse else {
      throw URLError(.badServerResponse)
    }
    return http.statusCode
  }
}
