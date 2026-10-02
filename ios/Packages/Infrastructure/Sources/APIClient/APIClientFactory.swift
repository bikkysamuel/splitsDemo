import OpenAPIRuntime
import OpenAPIURLSession

/// Builds the generated API client (ADR-0012) on the app's cache-less
/// `URLSession` (ADR-0005).
public enum APIClientFactory {
  /// - Parameter accessToken: the saved access token, read before each
  ///   protected request; `nil` sends none.
  public static func makeClient(
    configuration: APIConfiguration,
    accessToken: @escaping @Sendable () async -> String? = { nil }
  ) -> Client {
    Client(
      serverURL: configuration.baseURL,
      configuration: clientConfiguration,
      transport: URLSessionTransport(configuration: .init(session: URLSessionFactory.makeSession())),
      middlewares: [AuthenticationMiddleware(accessToken: accessToken)])
  }

  /// Shared by the app and tests, so both decode the server's dates alike.
  public static let clientConfiguration = Configuration(dateTranscoder: RFC3339DateTranscoder())
}
