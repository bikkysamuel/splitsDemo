import OpenAPIRuntime
import OpenAPIURLSession

/// Builds the generated API client (ADR-0012) on the app's cache-less
/// `URLSession` (ADR-0005).
public enum APIClientFactory {
  /// A client without the auth middleware, for anonymous operations only
  /// (health checks).
  public static func makeAnonymousClient(configuration: APIConfiguration) -> Client {
    Client(serverURL: configuration.baseURL, configuration: clientConfiguration, transport: makeTransport())
  }

  /// The app's transport, on its cache-less `URLSession`.
  public static func makeTransport() -> any ClientTransport {
    URLSessionTransport(configuration: .init(session: URLSessionFactory.makeSession()))
  }

  /// Shared by the app and tests, so both decode the server's dates alike.
  public static let clientConfiguration = Configuration(dateTranscoder: RFC3339DateTranscoder())
}
