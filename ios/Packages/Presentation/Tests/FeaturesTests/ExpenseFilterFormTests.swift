import Domain
import Foundation
import Testing

@testable import Features

struct ExpenseFilterFormTests {
  private let calendar = Calendar(identifier: .gregorian)
  private func day(_ s: String) -> Date { AddExpenseViewModel.date(s, calendar: calendar)! }

  @Test func readsBackTheFilterItWasMadeFrom() {
    let filter = ExpenseFilter(
      memberID: Group.me, category: .transport, from: "2026-10-01", to: "2026-10-31", state: .withdrawn)

    let form = ExpenseFilterForm(filter, calendar: calendar)

    #expect(form.filter(calendar: calendar) == filter)
  }

  @Test func datesCountOnlyWhenSwitchedOn() {
    var form = ExpenseFilterForm(.all, calendar: calendar)
    form.from = day("2026-10-05")
    #expect(form.filter(calendar: calendar) == .all)

    form.hasFrom = true
    #expect(form.filter(calendar: calendar) == ExpenseFilter(from: "2026-10-05"))
  }

  @Test func theLastDayIsNeverBeforeTheFirst() {
    var form = ExpenseFilterForm(.all, calendar: calendar)
    form.hasFrom = true
    form.hasTo = true
    form.from = day("2026-10-05")
    form.to = day("2026-10-01")

    #expect(form.filter(calendar: calendar).to == "2026-10-05")
  }
}
