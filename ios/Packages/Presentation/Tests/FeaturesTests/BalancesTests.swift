import Domain
import Foundation
import Testing

@testable import Features

actor FakeBalancesRepository: BalancesRepository {
  var result: [UUID: GroupBalances] = [:]
  private(set) var requested: [UUID] = []

  func set(_ balances: GroupBalances, for id: UUID) { result[id] = balances }

  func balances(groupID: UUID) async throws(ServiceError) -> GroupBalances {
    requested.append(groupID)
    guard let b = result[groupID] else { throw .problem(.notFound) }
    return b
  }
}

extension GroupBalances {
  static let tripOwes = GroupBalances(
    balances: [
      MemberBalance(memberID: Group.me, balance: Money(minorUnits: 5000, currency: "INR")),
      MemberBalance(memberID: Member.bob.id, balance: Money(minorUnits: -5000, currency: "INR")),
    ],
    suggestions: [
      SettleUpSuggestion(
        fromMemberID: Member.bob.id, toMemberID: Group.me, amount: Money(minorUnits: 5000, currency: "INR"))
    ])
}

/// Owed and owing read as words, so they don't depend on colour (NFR-A).
struct BalanceSentenceTests {
  private let inr = { (minor: Int64) in Money(minorUnits: minor, currency: "INR") }

  @Test func sayWhoIsOwedWhoOwesAndWhoIsSettled() {
    #expect(
      BalancesSection.sentence(MemberBalance(memberID: UUID(), balance: inr(5000)), name: "Alice").hasPrefix(
        "Alice is owed "))
    let owes = BalancesSection.sentence(MemberBalance(memberID: UUID(), balance: inr(-5000)), name: "Bob")
    #expect(owes.hasPrefix("Bob owes "))
    #expect(!owes.contains("-"))
    #expect(
      BalancesSection.sentence(MemberBalance(memberID: UUID(), balance: inr(0)), name: "Grandma")
        == "Grandma is settled up")
  }

  @Test func aSuggestionSaysWhoPaysWhom() {
    let s = SettleUpSuggestion(fromMemberID: UUID(), toMemberID: UUID(), amount: inr(5000))

    #expect(BalancesSection.suggestion(s, from: "Bob", to: "Alice").hasPrefix("Bob pays Alice "))
  }
}

@MainActor
struct ReportViewModelTests {
  private let flat = GroupSummary(id: UUID(), name: "Flat", currency: "EUR", state: .active)

  @Test func picksTheRememberedGroupAndLoadsItsBalances() async {
    let groups = FakeGroupsRepository()
    await groups.set(groups: .success([flat, Group.trip.summary]))
    let balances = FakeBalancesRepository()
    await balances.set(.tripOwes, for: Group.trip.id)
    let viewModel = ReportViewModel(
      groups: groups, balances: balances, preferences: FakePreferences(reportGroupID: Group.trip.id))

    await viewModel.load()

    #expect(viewModel.selectedGroupID == Group.trip.id)
    #expect(viewModel.report?.value?.balances == .tripOwes)
  }

  @Test func withoutARememberedGroupPicksTheFirst() async {
    let groups = FakeGroupsRepository()
    await groups.set(groups: .success([Group.trip.summary, flat]))
    let balances = FakeBalancesRepository()
    await balances.set(.tripOwes, for: Group.trip.id)
    let viewModel = ReportViewModel(groups: groups, balances: balances, preferences: FakePreferences())

    await viewModel.load()

    #expect(viewModel.selectedGroupID == Group.trip.id)
  }

  @Test func theChoiceIsRemembered() async {
    let preferences = FakePreferences()
    let viewModel = ReportViewModel(
      groups: FakeGroupsRepository(), balances: FakeBalancesRepository(), preferences: preferences)

    viewModel.selectedGroupID = flat.id

    #expect(preferences.reportGroupID == flat.id)
  }

  @Test func noGroupsMeansNoReport() async {
    let viewModel = ReportViewModel(
      groups: FakeGroupsRepository(), balances: FakeBalancesRepository(), preferences: FakePreferences())

    await viewModel.load()

    #expect(viewModel.groups == .loaded([]))
    #expect(viewModel.report == nil)
  }
}

@MainActor
struct GroupBalancesTests {
  @Test func theGroupScreenLoadsBalancesAndReloadsThemAfterAnExpense() async {
    let balances = FakeBalancesRepository()
    await balances.set(.tripOwes, for: Group.trip.id)
    let viewModel = GroupViewModel(
      groupID: Group.trip.id, repository: FakeGroupsRepository(), expenses: FakeExpensesRepository(), balances: balances
    )

    await viewModel.load()
    #expect(viewModel.balances == .tripOwes)

    await viewModel.expenseAdded()
    #expect(await balances.requested.count == 2)
  }
}
