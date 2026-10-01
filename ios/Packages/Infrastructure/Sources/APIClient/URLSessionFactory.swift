import Foundation

/// Builds the app's only `URLSession`.
public enum URLSessionFactory {
  /// An ephemeral session with no HTTP cache: the app keeps no domain data on
  /// the device, not even in the URL cache (ADR-0005).
  public static func makeSession() -> URLSession {
    let configuration = URLSessionConfiguration.ephemeral
    configuration.urlCache = nil
    configuration.requestCachePolicy = .reloadIgnoringLocalCacheData
    return URLSession(configuration: configuration)
  }
}
