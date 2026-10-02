import Domain

/// The repositories screens beyond sign-in use, built once by the
/// composition root (ADR-0015).
public struct AppDependencies: Sendable {
  public let groups: any GroupsRepository
  public let expenses: any ExpensesRepository
  public let balances: any BalancesRepository
  public let preferences: any PreferencesRepository

  public init(
    groups: any GroupsRepository, expenses: any ExpensesRepository, balances: any BalancesRepository,
    preferences: any PreferencesRepository
  ) {
    self.groups = groups
    self.expenses = expenses
    self.balances = balances
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
