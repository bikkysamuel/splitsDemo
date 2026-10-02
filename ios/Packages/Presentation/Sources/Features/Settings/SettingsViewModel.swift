import Domain
import Observation

/// Settings' preferences (FR-U5). The default currency pre-fills Create
/// Group and is kept in UserDefaults (ADR-0005).
@MainActor
@Observable
public final class SettingsViewModel {
  public var defaultCurrency: String {
    didSet { preferences.setDefaultCurrency(defaultCurrency) }
  }

  private let preferences: any PreferencesRepository

  public init(preferences: any PreferencesRepository) {
    self.preferences = preferences
    self.defaultCurrency = preferences.defaultCurrency()
  }
}
