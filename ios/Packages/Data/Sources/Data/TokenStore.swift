import Foundation

/// A Session's tokens as saved on the device. Only the Keychain holds them
/// (ADR-0005, doc 08).
public struct StoredTokens: Codable, Equatable, Sendable {
  public let accessToken: String
  public let accessExpiresAt: Date
  public let refreshToken: String
  public let refreshExpiresAt: Date

  public init(accessToken: String, accessExpiresAt: Date, refreshToken: String, refreshExpiresAt: Date) {
    self.accessToken = accessToken
    self.accessExpiresAt = accessExpiresAt
    self.refreshToken = refreshToken
    self.refreshExpiresAt = refreshExpiresAt
  }
}

/// Where the Session's tokens live.
public protocol TokenStore: Sendable {
  func load() async -> StoredTokens?
  func save(_ tokens: StoredTokens) async throws
  func clear() async
}

/// Keeps tokens in memory only: for tests and SwiftUI previews.
public actor InMemoryTokenStore: TokenStore {
  private var tokens: StoredTokens?

  public init(_ tokens: StoredTokens? = nil) {
    self.tokens = tokens
  }

  public func load() -> StoredTokens? { tokens }
  public func save(_ tokens: StoredTokens) { self.tokens = tokens }
  public func clear() { tokens = nil }
}
