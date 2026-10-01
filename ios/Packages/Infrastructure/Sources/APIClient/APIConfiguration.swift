import Foundation

/// Where the API lives. The base URL comes from the build configuration
/// through the `SplitsAPIBaseURL` Info.plist key (Q75).
public struct APIConfiguration: Sendable, Equatable {
  /// The Info.plist key holding the base URL.
  public static let baseURLKey = "SplitsAPIBaseURL"

  public let baseURL: URL

  /// Reads and validates the base URL. Pass `requiresHTTPS: true` in Release
  /// builds, which may only talk to the server over HTTPS (Q75).
  public init(infoDictionary: [String: Any], requiresHTTPS: Bool) throws(APIConfigurationError) {
    guard let value = infoDictionary[Self.baseURLKey] as? String else {
      throw .missingBaseURL
    }
    guard let url = URL(string: value), let scheme = url.scheme, url.host() != nil,
      scheme == "http" || scheme == "https"
    else {
      throw .invalidBaseURL(value)
    }
    if requiresHTTPS && scheme != "https" {
      throw .insecureBaseURL(value)
    }
    baseURL = url
  }
}

public enum APIConfigurationError: Error, Equatable {
  case missingBaseURL
  case invalidBaseURL(String)
  case insecureBaseURL(String)
}
