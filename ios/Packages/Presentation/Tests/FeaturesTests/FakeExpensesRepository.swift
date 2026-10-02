import Domain
import Foundation

/// A scripted `ExpensesRepository` for ViewModel tests.
actor FakeExpensesRepository: ExpensesRepository {
  var previewResult: Result<ExpensePreview, ServiceError>?
  var createResult: Result<Expense, ServiceError> = .success(.dinner)
  var pages: [String?: ExpensePage] = [nil: ExpensePage(items: [], nextCursor: nil)]
  var expenseResult: Result<Expense, ServiceError> = .success(.dinner)
  private(set) var previews: [ExpenseInput] = []
  private(set) var created: [(input: ExpenseInput, key: WriteKey)] = []

  func set(preview: Result<ExpensePreview, ServiceError>) { previewResult = preview }
  func set(create: Result<Expense, ServiceError>) { createResult = create }
  func set(pages: [String?: ExpensePage]) { self.pages = pages }

  func preview(groupID: UUID, input: ExpenseInput) async throws(ServiceError) -> ExpensePreview {
    previews.append(input)
    if let previewResult { return try previewResult.get() }
    // Equal split, leftover to the first: enough for the screens.
    let n = Int64(input.members.count)
    let each = input.amount.minorUnits / n
    let left = input.amount.minorUnits - each * n
    return ExpensePreview(
      amount: input.amount,
      shares: input.members.enumerated().map { i, id in
        Share(
          memberID: id, amount: Money(minorUnits: each + (Int64(i) < left ? 1 : 0), currency: input.amount.currency))
      })
  }

  func createExpense(groupID: UUID, input: ExpenseInput, key: WriteKey) async throws(ServiceError) -> Expense {
    created.append((input, key))
    return try createResult.get()
  }

  func expenses(groupID: UUID, cursor: String?) async throws(ServiceError) -> ExpensePage {
    pages[cursor] ?? ExpensePage(items: [], nextCursor: nil)
  }

  func expense(id: UUID) async throws(ServiceError) -> Expense { try expenseResult.get() }
}

extension Expense {
  static let dinner = Expense(
    id: UUID(), groupID: Group.trip.id, payerID: Group.me, createdByID: Group.me,
    amount: Money(minorUnits: 100001, currency: "INR"), category: .foodDrink, note: "Dinner", spentOn: "2026-10-01",
    state: .accepted, version: 1,
    shares: [Share(memberID: Group.me, amount: Money(minorUnits: 100001, currency: "INR"))])

  var summary: ExpenseSummary {
    ExpenseSummary(
      id: id, payerID: payerID, amount: amount, category: category, note: note, spentOn: spentOn, state: state)
  }
}
