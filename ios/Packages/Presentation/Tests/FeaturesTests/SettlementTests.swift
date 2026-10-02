import Domain
import Foundation
import Testing

@testable import Features

actor FakeSettlementsRepository: SettlementsRepository {
  var recordResults: [Result<Settlement, ServiceError>] = []
  var withdrawResult: Result<Settlement, ServiceError> = .success(Settlement.cash.withdrawn)
  var page = SettlementPage(items: [], nextCursor: nil)
  var current: Settlement = .cash
  private(set) var recorded: [(input: SettlementInput, acknowledged: Bool, key: WriteKey)] = []
  private(set) var withdrawals: [(version: Int, key: WriteKey)] = []

  func set(records: [Result<Settlement, ServiceError>]) { recordResults = records }
  func set(withdraw: Result<Settlement, ServiceError>) { withdrawResult = withdraw }
  func set(current: Settlement) { self.current = current }

  func record(
    groupID: UUID, input: SettlementInput, acknowledgeWarnings: Bool, key: WriteKey
  )
    async throws(ServiceError) -> Settlement
  {
    recorded.append((input, acknowledgeWarnings, key))
    return try recordResults.isEmpty ? Settlement.cash : recordResults.removeFirst().get()
  }

  func settlements(groupID: UUID, cursor: String?) async throws(ServiceError) -> SettlementPage { page }
  func settlement(id: UUID) async throws(ServiceError) -> Settlement { current }

  func withdraw(id: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Settlement {
    withdrawals.append((version, key))
    return try withdrawResult.get()
  }
}

extension Settlement {
  static let cash = Settlement(
    id: UUID(), groupID: Group.trip.id, fromMemberID: Member.bob.id, toMemberID: Group.me,
    amount: Money(minorUnits: 5000, currency: "INR"), settledOn: "2026-10-02", note: nil, createdByID: Group.me,
    state: .accepted, version: 1)

  var withdrawn: Settlement {
    Settlement(
      id: id, groupID: groupID, fromMemberID: fromMemberID, toMemberID: toMemberID, amount: amount,
      settledOn: settledOn, note: note, createdByID: createdByID, state: .withdrawn, version: version + 1)
  }

  func created(by member: UUID) -> Settlement {
    Settlement(
      id: id, groupID: groupID, fromMemberID: fromMemberID, toMemberID: toMemberID, amount: amount,
      settledOn: settledOn, note: note, createdByID: member, state: state, version: version)
  }
}

@MainActor
struct RecordSettlementViewModelTests {
  private let us = Locale(identifier: "en_US")

  @Test func aSuggestionPrefillsTheForm() {
    let s = SettleUpSuggestion(
      fromMemberID: Member.bob.id, toMemberID: Group.me, amount: Money(minorUnits: 33334, currency: "INR"))

    let viewModel = RecordSettlementViewModel(
      group: .withThree, repository: FakeSettlementsRepository(), suggestion: s, locale: us)

    #expect(viewModel.fromMemberID == Member.bob.id)
    #expect(viewModel.toMemberID == Group.me)
    #expect(viewModel.amountText == "333.34")
    #expect(viewModel.input?.amount == Money(minorUnits: 33334, currency: "INR"))
  }

  @Test func withoutASuggestionIPayTheFirstOtherMember() {
    let viewModel = RecordSettlementViewModel(group: .withThree, repository: FakeSettlementsRepository(), locale: us)

    #expect(viewModel.fromMemberID == Group.me)
    #expect(viewModel.toMemberID == Member.bob.id)
    #expect(viewModel.input == nil)
  }

  @Test func payingYourselfIsNotAnInput() {
    let viewModel = RecordSettlementViewModel(group: .withThree, repository: FakeSettlementsRepository(), locale: us)
    viewModel.amountText = "10"
    viewModel.toMemberID = Group.me

    #expect(viewModel.sameMember)
    #expect(!viewModel.canSubmit)
  }

  // FR-S2: an overpayment asks first; confirming resends with a new key.
  @Test func anOverpaymentAsksThenSavesWithANewKey() async {
    let repository = FakeSettlementsRepository()
    await repository.set(records: [.failure(.needsConfirmation(["overpayment"])), .success(.cash)])
    let viewModel = RecordSettlementViewModel(group: .withThree, repository: repository, locale: us)
    viewModel.amountText = "500"

    #expect(await viewModel.submit() == nil)
    #expect(viewModel.needsConfirmation)
    #expect(viewModel.errors == FormErrors())

    #expect(await viewModel.submit(acknowledge: true) == .cash)
    #expect(!viewModel.needsConfirmation)
    let recorded = await repository.recorded
    #expect(recorded.map(\.acknowledged) == [false, true])
    #expect(recorded[0].key != recorded[1].key)
  }

  @Test func refusedFieldsShow() async {
    let repository = FakeSettlementsRepository()
    await repository.set(records: [.failure(.invalidFields([FieldIssue(field: "to_member_id", reason: .sameMember)]))])
    let viewModel = RecordSettlementViewModel(group: .withThree, repository: repository, locale: us)
    viewModel.amountText = "5"

    _ = await viewModel.submit()

    #expect(viewModel.errors["to_member_id"] == "Choose someone other than the payer.")
  }
}

@MainActor
struct SettlementDetailViewModelTests {
  @Test func theCreatorMayWithdraw() async {
    let repository = FakeSettlementsRepository()
    let viewModel = SettlementDetailViewModel(
      settlementID: Settlement.cash.id, group: .withThree, repository: repository)
    await viewModel.load()

    #expect(viewModel.canWithdraw)
    #expect(await viewModel.withdraw())
    #expect(viewModel.state.value?.state == .withdrawn)
    #expect(await repository.withdrawals.map(\.version) == [1])
    #expect(!viewModel.canWithdraw)
  }

  @Test func othersMayNot() async {
    let repository = FakeSettlementsRepository()
    await repository.set(current: Settlement.cash.created(by: Member.bob.id))
    let viewModel = SettlementDetailViewModel(
      settlementID: Settlement.cash.id, group: .withThree, repository: repository)
    await viewModel.load()

    #expect(!viewModel.canWithdraw)
    #expect(await !viewModel.withdraw())
  }

  @Test func aRefusalIsShownAndReloads() async {
    let repository = FakeSettlementsRepository()
    await repository.set(withdraw: .failure(.problem(.versionConflict)))
    let viewModel = SettlementDetailViewModel(
      settlementID: Settlement.cash.id, group: .withThree, repository: repository)
    await viewModel.load()

    #expect(await !viewModel.withdraw())
    #expect(viewModel.actionError == "Someone else changed this just now. Reload and try again.")
  }
}
