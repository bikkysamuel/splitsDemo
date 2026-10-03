import Domain
import Foundation

/// A scripted `ExpensesRepository` for ViewModel tests.
actor FakeExpensesRepository: ExpensesRepository {
  var previewResult: Result<ExpensePreview, ServiceError>?
  var createResult: Result<Expense, ServiceError> = .success(.dinner)
  var pages: [String?: ExpensePage] = [nil: ExpensePage(items: [], nextCursor: nil)]
  var expenseResult: Result<Expense, ServiceError> = .success(.dinner)
  private(set) var previews: [ExpenseInput] = []
  /// Every list request: its filter and cursor.
  private(set) var listed: [(filter: ExpenseFilter, cursor: String?)] = []
  private(set) var created: [(input: ExpenseInput, key: WriteKey)] = []
  /// Answers to edits, in order; the last one repeats.
  var editResults: [Result<Expense, ServiceError>] = [.success(.dinner.edited)]
  var withdrawResult: Result<Expense, ServiceError> = .success(.dinner.withdrawn)
  private(set) var edits: [(id: UUID, version: Int, input: ExpenseInput, key: WriteKey)] = []
  private(set) var withdrawals: [(version: Int, key: WriteKey)] = []

  func set(edits: [Result<Expense, ServiceError>]) { editResults = edits }
  func set(withdraw: Result<Expense, ServiceError>) { withdrawResult = withdraw }
  func set(current: Expense) { expenseResult = .success(current) }

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
      shares: input.members.enumerated().map { i, entry in
        Share(
          memberID: entry.memberID,
          amount: Money(minorUnits: each + (Int64(i) < left ? 1 : 0), currency: input.amount.currency))
      })
  }

  func createExpense(groupID: UUID, input: ExpenseInput, key: WriteKey) async throws(ServiceError) -> Expense {
    created.append((input, key))
    return try createResult.get()
  }

  func expenses(groupID: UUID, filter: ExpenseFilter, cursor: String?) async throws(ServiceError) -> ExpensePage {
    listed.append((filter, cursor))
    return pages[cursor] ?? ExpensePage(items: [], nextCursor: nil)
  }

  func expense(id: UUID) async throws(ServiceError) -> Expense { try expenseResult.get() }

  func editExpense(
    id: UUID, version: Int, input: ExpenseInput, key: WriteKey
  ) async throws(ServiceError)
    -> Expense
  {
    edits.append((id, version, input, key))
    let result = editResults.count > 1 ? editResults.removeFirst() : editResults[0]
    return try result.get()
  }

  func withdrawExpense(id: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Expense {
    withdrawals.append((version, key))
    return try withdrawResult.get()
  }
}

extension Expense {
  static let dinner = Expense(
    id: UUID(), groupID: Group.trip.id, payerID: Group.me, createdByID: Group.me,
    amount: Money(minorUnits: 100001, currency: "INR"), category: .foodDrink, note: "Dinner", spentOn: "2026-10-01",
    state: .accepted, version: 1,
    shares: [Share(memberID: Group.me, amount: Money(minorUnits: 100001, currency: "INR"))])

  /// Dinner after an edit to ₹1,200: revision 2.
  var edited: Expense { changed(amount: 120000, revision: revision + 1) }

  var withdrawn: Expense { changed(state: .withdrawn) }

  func created(by memberID: UUID) -> Expense {
    Expense(
      id: id, groupID: groupID, payerID: payerID, createdByID: memberID, amount: amount, category: category, note: note,
      spentOn: spentOn, state: state, revision: revision, version: version, shares: shares)
  }

  func changed(amount minor: Int64? = nil, state: ExpenseState? = nil, revision: Int? = nil) -> Expense {
    let money = minor.map { Money(minorUnits: $0, currency: amount.currency) } ?? amount
    return Expense(
      id: id, groupID: groupID, payerID: payerID, createdByID: createdByID, amount: money,
      originalAmount: minor == nil ? originalAmount : money, exchangeRate: exchangeRate, category: category,
      note: note, spentOn: spentOn, state: state ?? self.state, revision: revision ?? self.revision,
      version: version + 1, splitMethod: splitMethod,
      shares: minor == nil ? shares : [Share(memberID: Group.me, amount: money)])
  }

  var summary: ExpenseSummary {
    ExpenseSummary(
      id: id, payerID: payerID, amount: amount, category: category, note: note, spentOn: spentOn, state: state)
  }
}
