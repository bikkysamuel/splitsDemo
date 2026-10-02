import Foundation
import Security

/// Keeps the Session's tokens in one Keychain item, readable after the
/// first unlock and never synced or restored to another device (doc 08:
/// `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`).
public struct KeychainTokenStore: TokenStore {
  private let service: String
  private let account = "session"

  /// - Parameter service: the Keychain service name; tests pass their own.
  public init(service: String = "dev.splits.session") {
    self.service = service
  }

  public func load() async -> StoredTokens? {
    var query = baseQuery
    query[kSecReturnData as String] = true
    query[kSecMatchLimit as String] = kSecMatchLimitOne
    var result: CFTypeRef?
    guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess, let data = result as? Data else {
      return nil
    }
    return try? JSONDecoder().decode(StoredTokens.self, from: data)
  }

  public func save(_ tokens: StoredTokens) async throws {
    let data = try JSONEncoder().encode(tokens)
    let attributes: [String: Any] = [
      kSecValueData as String: data,
      kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
    ]
    var status = SecItemUpdate(baseQuery as CFDictionary, attributes as CFDictionary)
    if status == errSecItemNotFound {
      status = SecItemAdd(baseQuery.merging(attributes) { $1 } as CFDictionary, nil)
    }
    guard status == errSecSuccess else { throw KeychainError(status: status) }
  }

  public func clear() async {
    SecItemDelete(baseQuery as CFDictionary)
  }

  private var baseQuery: [String: Any] {
    [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: service,
      kSecAttrAccount as String: account,
      kSecUseDataProtectionKeychain as String: true,
    ]
  }
}

/// A Keychain call failed with an `OSStatus`.
public struct KeychainError: Error, Equatable {
  public let status: OSStatus
}
