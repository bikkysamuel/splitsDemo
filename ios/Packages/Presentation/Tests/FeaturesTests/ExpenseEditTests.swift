import Domain
import Foundation
import Testing

@testable import Features

/// Editing an Expense (FR-E6): the Add Expense form, pre-filled, saving a
/// new revision with the version it changes.
@MainActor
struct EditExpenseViewModelTests {
  private let us = Locale(identifier: "en_US")
  private let de = Locale(identifier: "de_DE")

  /// US$10.50 at 83.25 paid by Bob, split by percentage between Alice and
  /// Bob, on 2026-09-30.
  private static let abroad = Expense(
    id: UUID(), groupID: Group.withThree.id, payerID: Member.bob.id, createdByID: Group.me,
    amount: Money(minorUnits: 87413, currency: "INR"), originalAmount: Money(minorUnits: 1050, currency: "USD"),
    exchangeRate: "83.25", category: .travel, note: "Taxi", spentOn: "2026-09-30", state: .accepted, revision: 1,
    version: 3, splitMethod: .percentage,
    shares: [
      Share(memberID: Group.me, amount: Money(minorUnits: 58275, currency: "INR"), input: "66.67"),
      Share(memberID: Member.bob.id, amount: Money(minorUnits: 29138, currency: "INR"), input: "33.33"),
    ])

  @Test func editingPrefillsTheFormWithTheExpense() {
    let viewModel = AddExpenseViewModel(
      editing: Self.abroad, group: .withThree, repository: FakeExpensesRepository(), locale: de)

    #expect(viewModel.isEditing)
    #expect(viewModel.amountText == "10,50" && viewModel.currency == "USD" && viewModel.rateText == "83,25")
    #expect(viewModel.payerID == Member.bob.id && viewModel.category == .travel && viewModel.note == "Taxi")
    #expect(AddExpenseViewModel.day(viewModel.spentOn) == "2026-09-30")
    #expect(viewModel.method == .percentage)
    #expect(viewModel.splitMembers == [Group.me, Member.bob.id])
    #expect(viewModel.entryTexts == [Group.me: "66,67", Member.bob.id: "33,33"])
    // Unchanged, it reads back to what is saved.
    #expect(
      viewModel.input
        == ExpenseInput(
          payerID: Member.bob.id, amount: Money(minorUnits: 1050, currency: "USD"), exchangeRate: "83.25",
          category: .travel, note: "Taxi", spentOn: "2026-09-30", method: .percentage,
          members: [SplitEntry(memberID: Group.me, input: "66.67"), SplitEntry(memberID: Member.bob.id, input: "33.33")]
        ))
  }

  @Test func savingSendsTheEditWithTheVersion() async {
    let repository = FakeExpensesRepository()
    let viewModel = AddExpenseViewModel(editing: .dinner, group: .withThree, repository: repository, locale: us)
    viewModel.amountText = "1200"

    let saved = await viewModel.submit()

    #expect(saved == Expense.dinner.edited)
    let edits = await repository.edits
    #expect(edits.map(\.version) == [1] && edits.map(\.id) == [Expense.dinner.id])
    #expect(edits.first?.input.amount == Money(minorUnits: 120000, currency: "INR"))
    #expect(await repository.created.isEmpty)
  }

  // NFR-R4: a stale copy is told in place, and Reload fetches the current
  // Expense into the form so the next save uses its version.
  @Test func aStaleEditIsShownInPlaceAndReloadRefillsTheForm() async {
    let repository = FakeExpensesRepository()
    let current = Expense.dinner.edited
    await repository.set(edits: [.failure(.problem(.versionConflict)), .success(current.edited)])
    await repository.set(current: current)
    let viewModel = AddExpenseViewModel(editing: .dinner, group: .withThree, repository: repository, locale: us)
    viewModel.amountText = "900"

    #expect(await viewModel.submit() == nil)
    #expect(viewModel.isStale)
    #expect(viewModel.errors.message == "Someone else changed this just now. Reload and try again.")

    await viewModel.reload()

    #expect(!viewModel.isStale && viewModel.errors == FormErrors())
    #expect(viewModel.amountText == "1200.00")
    viewModel.amountText = "900"
    _ = await viewModel.submit()
    let edits = await repository.edits
    #expect(edits.map(\.version) == [1, 2])
    #expect(edits[0].key != edits[1].key)
  }
}

@MainActor
struct ExpenseDetailActionsTests {
  @Test func theCreatorMayEditAndWithdraw() async {
    let repository = FakeExpensesRepository()
    let viewModel = ExpenseDetailViewModel(expenseID: Expense.dinner.id, group: .withThree, repository: repository)
    await viewModel.load()

    #expect(viewModel.canEdit && viewModel.canWithdraw)
    #expect(await viewModel.withdraw())
    #expect(viewModel.state.value?.state == .withdrawn)
    #expect(await repository.withdrawals.map(\.version) == [1])
    #expect(!viewModel.canEdit && !viewModel.canWithdraw)
  }

  @Test func othersMayNot() async {
    let repository = FakeExpensesRepository()
    await repository.set(current: Expense.dinner.created(by: Member.bob.id))
    let viewModel = ExpenseDetailViewModel(expenseID: Expense.dinner.id, group: .withThree, repository: repository)
    await viewModel.load()

    #expect(!viewModel.canEdit && !viewModel.canWithdraw)
    #expect(await !viewModel.withdraw())
    #expect(await repository.withdrawals.isEmpty)
  }

  @Test func nothingChangesInAClosedGroup() async {
    let closed = Group(
      id: Group.withThree.id, name: "Goa trip", currency: "INR", state: .closed, version: 1, myMemberID: Group.me,
      members: Group.withThree.members)
    let viewModel = ExpenseDetailViewModel(
      expenseID: Expense.dinner.id, group: closed, repository: FakeExpensesRepository())
    await viewModel.load()

    #expect(!viewModel.canEdit && !viewModel.canWithdraw)
  }

  @Test func aRefusedWithdrawalIsShownAndReloads() async {
    let repository = FakeExpensesRepository()
    await repository.set(withdraw: .failure(.problem(.versionConflict)))
    let viewModel = ExpenseDetailViewModel(expenseID: Expense.dinner.id, group: .withThree, repository: repository)
    await viewModel.load()
    await repository.set(current: Expense.dinner.edited)

    #expect(await !viewModel.withdraw())
    #expect(viewModel.actionError == "Someone else changed this just now. Reload and try again.")
    #expect(viewModel.state.value == Expense.dinner.edited)
  }

  @Test func anEditReplacesTheShownExpense() async {
    let viewModel = ExpenseDetailViewModel(
      expenseID: Expense.dinner.id, group: .withThree, repository: FakeExpensesRepository())
    await viewModel.load()

    viewModel.edited(Expense.dinner.edited)

    #expect(viewModel.state.value?.revision == 2)
  }
}
