import Domain

/// The repositories screens beyond sign-in use, built once by the
/// composition root (ADR-0015).
public struct AppDependencies: Sendable {
  public let groups: any GroupsRepository
  public let expenses: any ExpensesRepository
  public let balances: any BalancesRepository
  public let settlements: any SettlementsRepository
  public let preferences: any PreferencesRepository

  public init(
    groups: any GroupsRepository, expenses: any ExpensesRepository, balances: any BalancesRepository,
    settlements: any SettlementsRepository, preferences: any PreferencesRepository
  ) {
    self.groups = groups
    self.expenses = expenses
    self.balances = balances
    self.settlements = settlements
    self.preferences = preferences
  }
}

/// A screen's load state.
enum LoadState<Value: Equatable>: Equatable {
  case loading
  case loaded(Value)
  /// A String Catalog key.
  case failed(String)

  var value: Value? {
    if case .loaded(let v) = self { v } else { nil }
  }
}
