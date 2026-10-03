import Foundation

/// An amount of money one Member paid on behalf of some Members of a Group
/// (GLOSSARY: Expense).
public struct Expense: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let groupID: UUID
  public let payerID: UUID
  public let createdByID: UUID
  /// In the Group Currency: what the Shares divide.
  public let amount: Money
  /// The amount in the currency it was paid in (GLOSSARY: Original Amount);
  /// the same as `amount` when there was no conversion.
  public let originalAmount: Money
  /// The Exchange Rate entered, in the server's form ("83.25"); nil when the
  /// Expense was paid in the Group Currency.
  public let exchangeRate: String?
  public let category: Category
  public let note: String?
  /// The day the money was spent, as "yyyy-MM-dd" (a date, not a moment).
  public let spentOn: String
  public let state: ExpenseState
  /// 1, then one more per edit (FR-E6); the Activity History keeps each
  /// revision's changes.
  public let revision: Int
  public let version: Int
  public let splitMethod: SplitMethod
  /// Every Member in the Split, in joining order.
  public let shares: [Share]

  public init(
    id: UUID, groupID: UUID, payerID: UUID, createdByID: UUID, amount: Money, originalAmount: Money? = nil,
    exchangeRate: String? = nil, category: Category, note: String?, spentOn: String, state: ExpenseState,
    revision: Int = 1,
    version: Int, splitMethod: SplitMethod = .equal, shares: [Share]
  ) {
    self.id = id
    self.groupID = groupID
    self.payerID = payerID
    self.createdByID = createdByID
    self.amount = amount
    self.originalAmount = originalAmount ?? amount
    self.exchangeRate = exchangeRate
    self.category = category
    self.note = note
    self.spentOn = spentOn
    self.state = state
    self.revision = revision
    self.version = version
    self.splitMethod = splitMethod
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
  /// What was entered for the Member, in the server's form ("33.33" for a
  /// percentage, "2" for a ratio part); nil for an equal Split.
  public let input: String?

  public init(memberID: UUID, amount: Money, input: String? = nil) {
    self.memberID = memberID
    self.amount = amount
    self.input = input
  }
}

/// The fixed Category list (D7).
public enum Category: String, CaseIterable, Sendable {
  case foodDrink = "food_drink"
  case groceries, transport, accommodation, rent, utilities, entertainment, shopping, health, travel, other
}

/// Where an Expense is in its life (doc 06). In M1 every Expense is
/// accepted at once (D9).
/// Which Expenses a list shows (FR-E8). A nil field doesn't filter; the
/// fields combine.
public struct ExpenseFilter: Equatable, Hashable, Sendable {
  /// Expenses this Member paid or has a Share in.
  public var memberID: UUID?
  public var category: Category?
  /// The first and last day, "yyyy-MM-dd", both included.
  public var from: String?
  public var to: String?
  public var state: ExpenseState?

  public init(
    memberID: UUID? = nil, category: Category? = nil, from: String? = nil, to: String? = nil,
    state: ExpenseState? = nil
  ) {
    self.memberID = memberID
    self.category = category
    self.from = from
    self.to = to
    self.state = state
  }

  /// Every Expense.
  public static let all = ExpenseFilter()

  /// Whether the filter leaves any Expense out.
  public var isActive: Bool { self != .all }
}

public enum ExpenseState: String, Hashable, Sendable {
  case pending, accepted, disputed
  case withdrawalPending = "withdrawal_pending"
  case withdrawn
}

/// An Expense as entered (FR-E1, FR-E2, FR-E5): split by `method` among
/// `members`, in joining order. `amount` is the Original Amount; in another
/// currency than the Group Currency it comes with an `exchangeRate`, in the
/// server's form (see `ExchangeRate.input(from:locale:)`).
public struct ExpenseInput: Equatable, Hashable, Sendable {
  public var payerID: UUID
  public var amount: Money
  public var exchangeRate: String?
  public var category: Category
  public var note: String?
  public var spentOn: String
  public var method: SplitMethod
  public var members: [SplitEntry]

  public init(
    payerID: UUID, amount: Money, exchangeRate: String? = nil, category: Category, note: String?, spentOn: String,
    method: SplitMethod = .equal, members: [SplitEntry]
  ) {
    self.payerID = payerID
    self.amount = amount
    self.exchangeRate = exchangeRate
    self.category = category
    self.note = note
    self.spentOn = spentOn
    self.method = method
    self.members = members
  }
}

/// The Shares an Expense would get, in the Split's order (FR-E4), and the
/// amount in the Group Currency they divide.
public struct ExpensePreview: Equatable, Sendable {
  /// In the Group Currency, converted from `originalAmount` if need be.
  public let amount: Money
  public let originalAmount: Money
  public let exchangeRate: String?
  public let shares: [Share]

  public init(amount: Money, originalAmount: Money? = nil, exchangeRate: String? = nil, shares: [Share]) {
    self.amount = amount
    self.originalAmount = originalAmount ?? amount
    self.exchangeRate = exchangeRate
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
  /// One page of the Group's Expenses matching `filter`, newest first;
  /// pass the previous page's `nextCursor` with the same filter.
  func expenses(groupID: UUID, filter: ExpenseFilter, cursor: String?) async throws(ServiceError) -> ExpensePage
  func expense(id: UUID) async throws(ServiceError) -> Expense
  /// Saves the creator's edit of the Expense at `version` as its next
  /// revision (FR-E6); `.problem(.versionConflict)` if someone else changed
  /// it since.
  func editExpense(id: UUID, version: Int, input: ExpenseInput, key: WriteKey) async throws(ServiceError) -> Expense
  /// Withdraws the creator's Expense at `version` (FR-E6): it stays, but no
  /// longer counts toward Balances.
  func withdrawExpense(id: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Expense
}
