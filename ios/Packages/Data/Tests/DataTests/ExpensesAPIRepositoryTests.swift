import APIClient
import Domain
import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import Data

/// Expenses over the generated client, with JSON recorded from the server.
struct ExpensesAPIRepositoryTests {
  private static let groupID = UUID(uuidString: "01a0faca-9e5d-7e8a-b077-b767aee674ff")!
  private static let me = UUID(uuidString: "01a0faca-9e5d-7fa8-8b88-5ff7b2916763")!
  private static let grandma = UUID(uuidString: "01a0faca-9f91-73ea-8b78-6946f276b876")!
  private static let expenseID = UUID(uuidString: "01a0faca-a073-778e-aa29-5e4621c0afa3")!
  private static let base = "/v1/groups/01a0faca-9e5d-7e8a-b077-b767aee674ff"

  private let input = ExpenseInput(
    payerID: me, amount: Money(minorUnits: 100001, currency: "INR"), category: .foodDrink, note: "Dinner",
    spentOn: "2026-10-01", members: [SplitEntry(memberID: me), SplitEntry(memberID: grandma)])

  @Test func previewSendsTheInputAndMapsTheShares() async throws {
    let transport = try PathTransport(["POST \(Self.base)/expenses/preview": .fixture("expense-preview-200")])
    let repository = try makeRepository(transport)

    let preview = try await repository.preview(groupID: Self.groupID, input: input)

    #expect(preview.amount == Money(minorUnits: 100001, currency: "INR"))
    #expect(preview.shares.map(\.amount.minorUnits) == [50001, 50000])
    #expect(preview.shares.map(\.memberID) == [Self.me, Self.grandma])
    let sent = try #require(await transport.sent.first)
    #expect(sent.request.headerFields[HTTPField.Name("Idempotency-Key")!] == nil)
    let body = try #require(try JSONSerialization.jsonObject(with: sent.body) as? [String: Any])
    #expect(body["category"] as? String == "food_drink")
    #expect(body["spent_on"] as? String == "2026-10-01")
    #expect((body["amount"] as? [String: Any])?["minor"] as? Int == 100001)
    #expect((body["split"] as? [String: Any])?["method"] as? String == "equal")
  }

  // FR-E2: the method and each Member's entry go as decimal strings.
  @Test func previewSendsTheMethodAndEachEntry() async throws {
    let transport = try PathTransport(["POST \(Self.base)/expenses/preview": .fixture("expense-preview-200")])
    let repository = try makeRepository(transport)
    var percentages = input
    percentages.method = .percentage
    percentages.members = [
      SplitEntry(memberID: Self.me, input: "66.67"), SplitEntry(memberID: Self.grandma, input: "33.33"),
    ]

    _ = try await repository.preview(groupID: Self.groupID, input: percentages)

    let sent = try #require(await transport.sent.first)
    let body = try #require(try JSONSerialization.jsonObject(with: sent.body) as? [String: Any])
    let split = try #require(body["split"] as? [String: Any])
    #expect(split["method"] as? String == "percentage")
    let members = try #require(split["members"] as? [[String: Any]])
    #expect(members.map { $0["input"] as? String } == ["66.67", "33.33"])
    #expect(members.first?["member_id"] as? String == Self.me.uuidString.lowercased())
  }

  @Test func aSplitThatDoesNotAddUpIsAFieldIssueOnTheSplit() async throws {
    let transport = try PathTransport([
      "POST \(Self.base)/expenses/preview": .fixture("expense-preview-400-split", status: .badRequest, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.invalidFields([FieldIssue(field: "split", reason: .percentagesNot100)])) {
      try await repository.preview(groupID: Self.groupID, input: input)
    }
  }

  @Test func refusedFieldsAreFieldIssues() async throws {
    let transport = try PathTransport([
      "POST \(Self.base)/expenses/preview": .fixture("expense-preview-400", status: .badRequest, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(
      throws: ServiceError.invalidFields([
        FieldIssue(field: "amount/minor", reason: .notPositive),
        FieldIssue(field: "amount/currency", reason: .notGroupCurrency),
      ])
    ) {
      try await repository.preview(groupID: Self.groupID, input: input)
    }
  }

  @Test func createSendsTheKeyAndMapsTheExpense() async throws {
    let transport = try PathTransport([
      "POST \(Self.base)/expenses": .fixture("expense-create-201", status: .created)
    ])
    let repository = try makeRepository(transport)
    let key = WriteKey()

    let expense = try await repository.createExpense(groupID: Self.groupID, input: input, key: key)

    #expect(expense.id == Self.expenseID)
    #expect(expense.state == .accepted)
    #expect(expense.category == .foodDrink)
    #expect(expense.note == "Dinner")
    #expect(expense.spentOn == "2026-10-01")
    #expect(expense.shares.map(\.amount.minorUnits) == [50001, 50000])
    #expect(
      try #require(await transport.sent.first).request.headerFields[HTTPField.Name("Idempotency-Key")!]
        == key.value.uuidString)
  }

  @Test func listMapsTheSummaries() async throws {
    let transport = try PathTransport(["GET \(Self.base)/expenses?limit=50": .fixture("expenses-page")])
    let repository = try makeRepository(transport)

    let page = try await repository.expenses(groupID: Self.groupID, cursor: nil)

    #expect(page.items.map(\.id) == [Self.expenseID])
    #expect(page.items.first?.amount == Money(minorUnits: 100001, currency: "INR"))
    #expect(page.nextCursor == nil)
  }

  @Test func getMapsTheExpense() async throws {
    let transport = try PathTransport([
      "GET /v1/expenses/\(Self.expenseID.uuidString.lowercased())": .fixture("expense-get-200")
    ])
    let repository = try makeRepository(transport)

    let expense = try await repository.expense(id: Self.expenseID)

    #expect(expense.payerID == Self.me)
    #expect(expense.shares.count == 2)
    #expect(expense.splitMethod == .equal)
  }

  @Test func getMapsTheSplitMethodAndEntries() async throws {
    let transport = try PathTransport([
      "GET /v1/expenses/\(Self.expenseID.uuidString.lowercased())": .fixture("expense-ratio-get-200")
    ])
    let repository = try makeRepository(transport)

    let expense = try await repository.expense(id: Self.expenseID)

    #expect(expense.splitMethod == .ratio)
    #expect(expense.shares.map(\.input) == ["2", "1"])
    #expect(expense.shares.map(\.amount.minorUnits) == [667, 334])
  }

  private func makeRepository(_ transport: PathTransport) throws -> ExpensesAPIRepository {
    ExpensesAPIRepository(
      api: APISession(
        serverURL: try #require(URL(string: "http://localhost:8080")), transport: transport,
        tokens: InMemoryTokenStore(.sample)))
  }
}

/// Balances come from the server as signed Money (FR-B1).
struct BalancesAPIRepositoryTests {
  @Test func mapsBalancesAndSuggestions() async throws {
    let me = try #require(UUID(uuidString: "01a0faca-9e5d-7fa8-8b88-5ff7b2916763"))
    let grandma = try #require(UUID(uuidString: "01a0faca-9f91-73ea-8b78-6946f276b876"))
    let transport = try PathTransport([
      "GET /v1/groups/01a0faca-9e5d-7e8a-b077-b767aee674ff/balances": .fixture("balances-200")
    ])
    let repository = BalancesAPIRepository(
      api: APISession(
        serverURL: try #require(URL(string: "http://localhost:8080")), transport: transport,
        tokens: InMemoryTokenStore(.sample)))

    let b = try await repository.balances(
      groupID: try #require(UUID(uuidString: "01a0faca-9e5d-7e8a-b077-b767aee674ff")))

    #expect(b.balances.map(\.direction) == [.owed, .owes])
    #expect(b.balances.map(\.balance.minorUnits) == [66667, -66667])
    #expect(
      b.suggestions == [
        SettleUpSuggestion(fromMemberID: grandma, toMemberID: me, amount: Money(minorUnits: 66667, currency: "INR"))
      ])
  }
}
