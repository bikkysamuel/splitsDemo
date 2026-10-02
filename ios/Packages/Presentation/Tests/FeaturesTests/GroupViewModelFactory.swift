import Domain
import Foundation

@testable import Features

extension GroupViewModel {
  /// A GroupViewModel on fakes; pass only the ones a test scripts.
  static func make(
    groupID: UUID = Group.trip.id, groups: FakeGroupsRepository = FakeGroupsRepository(),
    expenses: FakeExpensesRepository = FakeExpensesRepository(),
    balances: FakeBalancesRepository = FakeBalancesRepository(),
    settlements: FakeSettlementsRepository = FakeSettlementsRepository()
  ) -> GroupViewModel {
    GroupViewModel(
      groupID: groupID, repository: groups, expenses: expenses, balances: balances, settlements: settlements)
  }
}
