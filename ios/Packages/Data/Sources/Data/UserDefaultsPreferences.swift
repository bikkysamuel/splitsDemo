import Domain
import Foundation

/// Non-financial UI preferences in `UserDefaults` (ADR-0005, FR-U5).
public struct UserDefaultsPreferences: PreferencesRepository, @unchecked Sendable {
  // UserDefaults is thread-safe; it just isn't marked Sendable.
  private let defaults: UserDefaults
  private let locale: Locale

  static let defaultCurrencyKey = "defaultCurrency"

  public init(defaults: UserDefaults = .standard, locale: Locale = .current) {
    self.defaults = defaults
    self.locale = locale
  }

  /// The chosen currency, else the region's if the server accepts it, else
  /// USD.
  public func defaultCurrency() -> String {
    if let chosen = defaults.string(forKey: Self.defaultCurrencyKey), Currency.isActive(chosen) { return chosen }
    if let regional = locale.currency?.identifier, Currency.isActive(regional) { return regional }
    return "USD"
  }

  public func setDefaultCurrency(_ code: String) {
    defaults.set(code, forKey: Self.defaultCurrencyKey)
  }

  static let lastReportGroupKey = "lastReportGroupID"

  public func lastReportGroupID() -> UUID? {
    defaults.string(forKey: Self.lastReportGroupKey).flatMap(UUID.init(uuidString:))
  }

  public func setLastReportGroupID(_ id: UUID) {
    defaults.set(id.uuidString, forKey: Self.lastReportGroupKey)
  }
}
