import APIClient
import Foundation
import OpenAPIRuntime

/// The app's one connection to the API: the token store, the single
/// `SessionRefresher` and the client whose middleware attaches and refreshes
/// the access token. Every repository shares it, because two refreshers
/// refreshing at once would trip the server's reuse detection and end the
/// Session (ADR-0011).
public final class APISession: Sendable {
  let client: Client
  let tokens: any TokenStore
  let refresher: SessionRefresher

  public convenience init(configuration: APIConfiguration, tokens: any TokenStore) {
    self.init(serverURL: configuration.baseURL, transport: APIClientFactory.makeTransport(), tokens: tokens)
  }

  /// For tests: the same session over another transport.
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
}
