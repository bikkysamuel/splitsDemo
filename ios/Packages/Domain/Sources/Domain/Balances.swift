import Foundation

/// A Group's Balances and Settle-up Suggestions, computed by the server
/// (FR-B1, FR-B2).
public struct GroupBalances: Equatable, Sendable {
  /// Every Member, Former Members included, in joining order.
  public let balances: [MemberBalance]
  public let suggestions: [SettleUpSuggestion]

  public init(balances: [MemberBalance], suggestions: [SettleUpSuggestion]) {
    self.balances = balances
    self.suggestions = suggestions
  }
}

/// A Member's net position (GLOSSARY: Balance).
public struct MemberBalance: Equatable, Hashable, Sendable {
  public let memberID: UUID
  /// Positive: the Member is owed money; negative: they owe.
  public let balance: Money

  public init(memberID: UUID, balance: Money) {
    self.memberID = memberID
    self.balance = balance
  }

  public var direction: Direction {
    if balance.minorUnits > 0 { .owed } else if balance.minorUnits < 0 { .owes } else { .settled }
  }

  public enum Direction: Sendable { case owed, owes, settled }
}

/// A payment that helps settle the Group (GLOSSARY: Settle-up Suggestion).
public struct SettleUpSuggestion: Equatable, Hashable, Sendable {
  public let fromMemberID: UUID
  public let toMemberID: UUID
  public let amount: Money

  public init(fromMemberID: UUID, toMemberID: UUID, amount: Money) {
    self.fromMemberID = fromMemberID
    self.toMemberID = toMemberID
    self.amount = amount
  }
}

/// Balances (FR-B*).
public protocol BalancesRepository: Sendable {
  func balances(groupID: UUID) async throws(ServiceError) -> GroupBalances
}
