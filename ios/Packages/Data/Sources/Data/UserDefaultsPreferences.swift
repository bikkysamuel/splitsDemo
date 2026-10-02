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

  /// The chosen currency, else the region's, else USD.
  public func defaultCurrency() -> String {
    defaults.string(forKey: Self.defaultCurrencyKey) ?? locale.currency?.identifier ?? "USD"
  }

  public func setDefaultCurrency(_ code: String) {
    defaults.set(code, forKey: Self.defaultCurrencyKey)
  }
}
