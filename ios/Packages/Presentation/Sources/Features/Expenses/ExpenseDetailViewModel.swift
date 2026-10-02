import Domain
import Foundation
import Observation

/// An Expense with its Shares.
@MainActor
@Observable
public final class ExpenseDetailViewModel {
  public let expenseID: UUID
  /// For Member names.
  public let group: Domain.Group
  private(set) var state: LoadState<Expense> = .loading

  private let repository: any ExpensesRepository

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

  /// The Member's display name; nil if the Group no longer lists them.
  func name(_ memberID: UUID) -> String? { group.member(memberID)?.displayName }
}
