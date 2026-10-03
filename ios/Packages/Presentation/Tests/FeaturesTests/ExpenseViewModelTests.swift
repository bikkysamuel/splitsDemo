import Domain
import Foundation
import Testing

@testable import Features

@MainActor
struct AddExpenseViewModelTests {
  private let group = Group.trip.with(name: "Goa trip", version: 1)
  private let us = Locale(identifier: "en_US")

  private func makeViewModel(_ repository: FakeExpensesRepository) -> AddExpenseViewModel {
    let today = Calendar.current.date(from: DateComponents(year: 2026, month: 10, day: 1))!
    return AddExpenseViewModel(group: Group.withThree, repository: repository, today: today, locale: us)
  }

  @Test func startsWithMePayingAndEveryActiveMemberSharing() {
    let viewModel = makeViewModel(FakeExpensesRepository())

    #expect(viewModel.payerID == Group.me)
    #expect(viewModel.splitMembers == Set(Group.withThree.activeMembers.map(\.id)))
    #expect(viewModel.input == nil)
    #expect(!viewModel.canSubmit)
  }

  @Test func theInputUsesMinorUnitsOfTheGroupCurrencyAndJoinOrder() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "1,000.01"
    viewModel.note = "  Dinner "

    let input = viewModel.input

    #expect(input?.amount == Money(minorUnits: 100001, currency: "INR"))
    #expect(input?.note == "Dinner")
    #expect(input?.spentOn == "2026-10-01")
    #expect(input?.method == .equal)
    #expect(input?.members == Group.withThree.activeMembers.map { SplitEntry(memberID: $0.id) })
  }

  @Test func anUnreadableAmountIsFlaggedAndNotSent() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "12.345"

    #expect(viewModel.amountIsInvalid)
    #expect(viewModel.input == nil)
  }

  @Test func nobodySharingMeansNoInput() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "10"
    for m in Group.withThree.activeMembers { viewModel.toggle(m.id) }

    #expect(viewModel.input == nil)
  }

  // FR-E4: the preview's Shares come from the server.
  @Test func thePreviewShowsTheServersShares() async {
    let repository = FakeExpensesRepository()
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "1000.01"

    await viewModel.refreshPreview()

    #expect(viewModel.preview?.shares.map(\.amount.minorUnits) == [33334, 33334, 33333])
    #expect(await repository.previews.count == 1)
  }

  @Test func aRefusedPreviewIsShown() async {
    let repository = FakeExpensesRepository()
    await repository.set(preview: .failure(.problem(.notFound)))
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "10"

    await viewModel.refreshPreview()

    #expect(viewModel.preview == nil)
    #expect(viewModel.previewError == ServiceErrorMessage.key(for: .notFound))
  }

  @Test func savingSendsTheInputAndReturnsTheExpense() async {
    let repository = FakeExpensesRepository()
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "1000.01"

    #expect(await viewModel.submit() == .dinner)
    #expect(await repository.created.count == 1)
  }

  // NFR-R1: retrying the same Expense reuses its key.
  @Test func aRetryReusesTheIdempotencyKey() async {
    let repository = FakeExpensesRepository()
    await repository.set(create: .failure(.unreachable))
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "10"

    _ = await viewModel.submit()
    _ = await viewModel.submit()

    let keys = await repository.created.map(\.key)
    #expect(keys.count == 2 && keys[0] == keys[1])
    #expect(viewModel.errors.message == ServiceErrorMessage.unreachable)
  }

  // FR-E2: every sharing Member gets an entry, sent in the server's form.
  @Test func aPercentageSplitSendsEachSharingMembersEntry() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "100"
    viewModel.method = .percentage
    viewModel.toggle(Member.grandma.id)
    viewModel.entryTexts = [Group.me: "66.67", Member.bob.id: "33.33", Member.grandma.id: "10"]

    #expect(viewModel.input?.method == .percentage)
    #expect(
      viewModel.input?.members == [
        SplitEntry(memberID: Group.me, input: "66.67"), SplitEntry(memberID: Member.bob.id, input: "33.33"),
      ])
  }

  @Test func anExactSplitSendsMinorUnits() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "10"
    viewModel.method = .exact
    viewModel.entryTexts = [Group.me: "2.50", Member.bob.id: "7", Member.grandma.id: "0.50"]

    #expect(viewModel.input?.members.map(\.input) == ["250", "700", "50"])
  }

  @Test func aRatioSplitSendsWholeParts() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "10"
    viewModel.method = .ratio
    viewModel.entryTexts = [Group.me: "2", Member.bob.id: "1", Member.grandma.id: " 1 "]

    #expect(viewModel.input?.method == .ratio)
    #expect(viewModel.input?.members.map(\.input) == ["2", "1", "1"])
  }

  // Form validation checks well-formedness only; sums are the server's.
  @Test func aMissingOrMalformedEntryMeansNoInput() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "10"
    viewModel.method = .ratio
    viewModel.entryTexts = [Group.me: "2", Member.bob.id: "1.5"]

    #expect(viewModel.input == nil)
    #expect(viewModel.entryIsInvalid(Member.bob.id))
    #expect(!viewModel.entryIsInvalid(Member.grandma.id))  // blank, not wrong yet
    #expect(!viewModel.entryIsInvalid(Group.me))
  }

  @Test func changingTheMethodClearsTheEntries() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.method = .ratio
    viewModel.entryTexts = [Group.me: "2"]

    viewModel.method = .percentage

    #expect(viewModel.entryTexts.isEmpty)
  }

  @Test func aSplitThatDoesNotAddUpIsShownUnderTheSplit() async {
    let repository = FakeExpensesRepository()
    await repository.set(preview: .failure(.invalidFields([FieldIssue(field: "split", reason: .percentagesNot100)])))
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "100"
    viewModel.method = .percentage
    viewModel.entryTexts = [Group.me: "50", Member.bob.id: "40", Member.grandma.id: "5"]

    await viewModel.refreshPreview()

    #expect(viewModel.splitError == "The percentages must add up to exactly 100.")
    #expect(viewModel.previewError == nil)
  }

  @Test func aRefusedEntryIsShownForItsMember() async {
    let repository = FakeExpensesRepository()
    await repository.set(
      preview: .failure(.invalidFields([FieldIssue(field: "split/members/1/input", reason: .tooManyDecimals)])))
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "100"
    viewModel.method = .ratio
    viewModel.toggle(Group.me)
    viewModel.entryTexts = [Member.bob.id: "1", Member.grandma.id: "1"]

    await viewModel.refreshPreview()

    // Index 1 of the Split is Grandma, since Alice isn't sharing.
    #expect(viewModel.entryError(for: Member.grandma.id) == "Use at most 2 decimal places.")
    #expect(viewModel.entryError(for: Member.bob.id) == nil)
  }

  // A refused save's entry error goes once the next preview answers: it
  // points at a Split index that may no longer be the same Member.
  @Test func theNextPreviewReplacesASavesFieldErrors() async {
    let repository = FakeExpensesRepository()
    await repository.set(
      create: .failure(.invalidFields([FieldIssue(field: "split/members/1/input", reason: .ratioNotInteger)])))
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "10"
    viewModel.method = .ratio
    viewModel.entryTexts = [Group.me: "2", Member.bob.id: "1", Member.grandma.id: "1"]
    _ = await viewModel.submit()
    #expect(viewModel.entryError(for: Member.bob.id) == "Use a whole number.")

    viewModel.toggle(Group.me)
    await viewModel.refreshPreview()

    #expect(viewModel.entryError(for: Member.grandma.id) == nil)
    #expect(viewModel.entryError(for: Member.bob.id) == nil)
  }

  @Test func refusedFieldsShowUnderTheField() async {
    let repository = FakeExpensesRepository()
    await repository.set(create: .failure(.invalidFields([FieldIssue(field: "note", reason: .tooLong)])))
    let viewModel = makeViewModel(repository)
    viewModel.amountText = "10"

    _ = await viewModel.submit()

    #expect(viewModel.errors["note"] == "Use at most 500 characters.")
  }

  // --- Another currency (FR-E5) ---

  @Test func startsInTheGroupCurrencyWithoutARate() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.amountText = "10"
    viewModel.rateText = "83.25"

    #expect(viewModel.currency == "INR")
    #expect(!viewModel.needsRate)
    #expect(viewModel.input?.exchangeRate == nil)
  }

  @Test func anotherCurrencySendsTheOriginalAmountWithItsRate() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.currency = "USD"
    viewModel.amountText = "10.50"
    #expect(viewModel.needsRate)
    #expect(viewModel.input == nil)

    viewModel.rateText = "83.25"

    #expect(viewModel.input?.amount == Money(minorUnits: 1050, currency: "USD"))
    #expect(viewModel.input?.exchangeRate == "83.25")
  }

  @Test func anUnreadableRateIsFlaggedAndNotSent() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.currency = "USD"
    viewModel.amountText = "10"
    viewModel.rateText = "83.255"

    #expect(viewModel.rateIsInvalid)
    #expect(viewModel.input == nil)
  }

  @Test func theAmountIsReadInTheChosenCurrency() {
    let viewModel = makeViewModel(FakeExpensesRepository())
    viewModel.currency = "JPY"
    viewModel.rateText = "0.55"
    viewModel.amountText = "1000.5"
    #expect(viewModel.amountIsInvalid)

    viewModel.amountText = "1000"

    #expect(viewModel.input?.amount == Money(minorUnits: 1000, currency: "JPY"))
  }

  // ADR-0006: the converted amount comes from the server.
  @Test func thePreviewShowsTheConvertedAmount() async {
    let repository = FakeExpensesRepository()
    await repository.set(
      preview: .success(
        ExpensePreview(
          amount: Money(minorUnits: 87413, currency: "INR"), originalAmount: Money(minorUnits: 1050, currency: "USD"),
          exchangeRate: "83.25", shares: [])))
    let viewModel = makeViewModel(repository)
    viewModel.currency = "USD"
    viewModel.amountText = "10.50"
    viewModel.rateText = "83.25"

    await viewModel.refreshPreview()

    #expect(viewModel.convertedAmount == Money(minorUnits: 87413, currency: "INR"))
  }

  @Test func aRefusedRateIsShownUnderTheRate() async {
    let repository = FakeExpensesRepository()
    await repository.set(
      preview: .failure(.invalidFields([FieldIssue(field: "amount/minor", reason: .convertsToZero)])))
    let viewModel = makeViewModel(repository)
    viewModel.currency = "USD"
    viewModel.amountText = "0.01"
    viewModel.rateText = "0.49"

    await viewModel.refreshPreview()

    #expect(
      viewModel.fieldError("amount/minor") == "This converts to less than the smallest unit of the Group Currency.")
    #expect(viewModel.previewError == nil)
  }
}

@MainActor
struct GroupExpensesTests {
  @Test func loadsTheFirstPageAndMore() async {
    let expenses = FakeExpensesRepository()
    let a = Expense.dinner.summary
    let b = ExpenseSummary(
      id: UUID(), payerID: Group.me, amount: Money(minorUnits: 500, currency: "INR"), category: .transport, note: nil,
      spentOn: "2026-09-30", state: .accepted)
    await expenses.set(pages: [
      nil: ExpensePage(items: [a], nextCursor: "c1"), "c1": ExpensePage(items: [b], nextCursor: nil),
    ])
    let viewModel = GroupViewModel.make(
      groupID: Group.trip.id, groups: FakeGroupsRepository(), expenses: expenses, balances: FakeBalancesRepository()
    )

    await viewModel.load()
    #expect(viewModel.expenses == [a])
    #expect(viewModel.hasMoreExpenses)

    await viewModel.loadMoreExpenses()
    #expect(viewModel.expenses == [a, b])
    #expect(!viewModel.hasMoreExpenses)
  }
}

@MainActor
struct ExpenseDetailViewModelTests {
  @Test func loadsTheExpenseAndNamesItsMembers() async {
    let viewModel = ExpenseDetailViewModel(
      expenseID: Expense.dinner.id, group: .trip, repository: FakeExpensesRepository())

    await viewModel.load()

    #expect(viewModel.state == .loaded(.dinner))
    #expect(viewModel.name(Group.me) == "Alice")
    #expect(viewModel.name(UUID()) == nil)
  }
}

extension Group {
  /// Alice (me), Bob and the Placeholder Grandma.
  static let withThree = Group(
    id: trip.id, name: "Goa trip", currency: "INR", state: .active, version: 1, myMemberID: me,
    members: [
      Member(id: me, displayName: "Alice", role: .admin, status: .active, isPlaceholder: false, joinSeq: 1),
      Member.bob, Member.grandma,
    ])
}
