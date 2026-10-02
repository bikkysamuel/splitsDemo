import Foundation

/// An amount of money one Member paid on behalf of some Members of a Group
/// (GLOSSARY: Expense).
public struct Expense: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let groupID: UUID
  public let payerID: UUID
  public let createdByID: UUID
  public let amount: Money
  public let category: Category
  public let note: String?
  /// The day the money was spent, as "yyyy-MM-dd" (a date, not a moment).
  public let spentOn: String
  public let state: ExpenseState
  public let version: Int
  /// Every Member in the Split, in joining order.
  public let shares: [Share]

  public init(
    id: UUID, groupID: UUID, payerID: UUID, createdByID: UUID, amount: Money, category: Category, note: String?,
    spentOn: String, state: ExpenseState, version: Int, shares: [Share]
  ) {
    self.id = id
    self.groupID = groupID
    self.payerID = payerID
    self.createdByID = createdByID
    self.amount = amount
    self.category = category
    self.note = note
    self.spentOn = spentOn
    self.state = state
    self.version = version
    self.shares = shares
  }
}

/// An Expense in a list.
public struct ExpenseSummary: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let payerID: UUID
  public let amount: Money
  public let category: Category
  public let note: String?
  public let spentOn: String
  public let state: ExpenseState

  public init(
    id: UUID, payerID: UUID, amount: Money, category: Category, note: String?, spentOn: String, state: ExpenseState
  ) {
    self.id = id
    self.payerID = payerID
    self.amount = amount
    self.category = category
    self.note = note
    self.spentOn = spentOn
    self.state = state
  }
}

/// The portion of an Expense one Member owes (GLOSSARY: Share).
public struct Share: Equatable, Hashable, Sendable {
  public let memberID: UUID
  public let amount: Money

  public init(memberID: UUID, amount: Money) {
    self.memberID = memberID
    self.amount = amount
  }
}

/// The fixed Category list (D7).
public enum Category: String, CaseIterable, Sendable {
  case foodDrink = "food_drink"
  case groceries, transport, accommodation, rent, utilities, entertainment, shopping, health, travel, other
}

/// Where an Expense is in its life (doc 06). In M1 every Expense is
/// accepted at once (D9).
public enum ExpenseState: String, Sendable {
  case pending, accepted, disputed
  case withdrawalPending = "withdrawal_pending"
  case withdrawn
}

/// An Expense as entered (FR-E1): split equally among `members` for now.
public struct ExpenseInput: Equatable, Hashable, Sendable {
  public var payerID: UUID
  public var amount: Money
  public var category: Category
  public var note: String?
  public var spentOn: String
  public var members: [UUID]

  public init(payerID: UUID, amount: Money, category: Category, note: String?, spentOn: String, members: [UUID]) {
    self.payerID = payerID
    self.amount = amount
    self.category = category
    self.note = note
    self.spentOn = spentOn
    self.members = members
  }
}

/// The Shares an Expense would get, in the Split's order (FR-E4).
public struct ExpensePreview: Equatable, Sendable {
  public let amount: Money
  public let shares: [Share]

  public init(amount: Money, shares: [Share]) {
    self.amount = amount
    self.shares = shares
  }
}

/// One page of a Group's Expenses, newest first.
public struct ExpensePage: Equatable, Sendable {
  public let items: [ExpenseSummary]
  /// Pass to fetch the next page; nil on the last.
  public let nextCursor: String?

  public init(items: [ExpenseSummary], nextCursor: String?) {
    self.items = items
    self.nextCursor = nextCursor
  }
}

/// Expenses (FR-E*). Shares always come from the server (ADR-0006).
public protocol ExpensesRepository: Sendable {
  /// The Shares the input would get, without saving.
  func preview(groupID: UUID, input: ExpenseInput) async throws(ServiceError) -> ExpensePreview
  func createExpense(groupID: UUID, input: ExpenseInput, key: WriteKey) async throws(ServiceError) -> Expense
  func expenses(groupID: UUID, cursor: String?) async throws(ServiceError) -> ExpensePage
  func expense(id: UUID) async throws(ServiceError) -> Expense
}
