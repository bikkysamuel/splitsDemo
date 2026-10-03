import Domain
import Foundation
import Observation

/// An Expense with its Shares, with Edit and Withdraw for its creator
/// (FR-E6).
@MainActor
@Observable
public final class ExpenseDetailViewModel {
  public let expenseID: UUID
  /// For Member names.
  public let group: Domain.Group
  private(set) var state: LoadState<Expense> = .loading
  /// Why the last withdrawal failed (a catalog key).
  private(set) var actionError: String?
  public private(set) var isWithdrawing = false

  /// The Repository the edit form saves through.
  let repository: any ExpensesRepository
  private var keys = WriteKeys<Int>()

  public init(expenseID: UUID, group: Domain.Group, repository: any ExpensesRepository) {
    self.expenseID = expenseID
    self.group = group
    self.repository = repository
  }

  public func load() async {
    do {
      state = .loaded(try await repository.expense(id: expenseID))
    } catch {
      state = .failed(ServiceErrorMessage.key(for: error))
    }
  }

  /// Only the creator changes an Expense, never a withdrawn one, nor in a
  /// Closed Group.
  private var isMineToChange: Bool {
    guard let e = state.value else { return false }
    return e.createdByID == group.myMemberID && group.state != .closed
      && e.state != .withdrawn && e.state != .withdrawalPending
  }

  var canEdit: Bool { isMineToChange }
  var canWithdraw: Bool { isMineToChange }

  /// Shows the Expense as the edit saved it.
  func edited(_ expense: Expense) { state = .loaded(expense) }

  /// Withdraws the Expense; returns true on success. On a refusal the
  /// Expense is reloaded, so what is shown is current.
  public func withdraw() async -> Bool {
    guard let e = state.value, canWithdraw, !isWithdrawing else { return false }
    isWithdrawing = true
    defer { isWithdrawing = false }
    do {
      state = .loaded(try await repository.withdrawExpense(id: e.id, version: e.version, key: keys.key(for: e.version)))
      keys.succeeded()
      actionError = nil
      return true
    } catch {
      actionError = ServiceErrorMessage.key(for: error)
      await load()
      return false
    }
  }

  func dismissError() { actionError = nil }

  /// The Member's display name; nil if the Group no longer lists them.
  func name(_ memberID: UUID) -> String? { group.member(memberID)?.displayName }
}
