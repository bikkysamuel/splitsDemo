import Foundation
import Testing

@testable import Domain

struct ExpenseFilterTests {
  @Test func noFilterShowsEverything() {
    #expect(!ExpenseFilter.all.isActive)
    #expect(ExpenseFilter(category: .transport).isActive)
    #expect(ExpenseFilter(from: "2026-10-01").isActive)
  }
}
