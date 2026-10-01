import OpenAPIURLSession

/// Builds the generated API client (ADR-0012) on the app's cache-less
/// `URLSession` (ADR-0005).
public enum APIClientFactory {
  public static func makeClient(configuration: APIConfiguration) -> Client {
    Client(
      serverURL: configuration.baseURL,
      transport: URLSessionTransport(configuration: .init(session: URLSessionFactory.makeSession())))
  }
}
