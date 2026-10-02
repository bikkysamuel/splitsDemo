import APIClient
import Domain
import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import Data

/// Settlements over the generated client, with JSON recorded from the server.
struct SettlementsAPIRepositoryTests {
  private static let groupID = UUID(uuidString: "01a0fae1-4f3a-7c92-8675-9d81e1b3ac9e")!
  private static let me = UUID(uuidString: "01a0fae1-4f3a-7c92-8675-9d81e1b3ac9f")!
  private static let grandma = UUID(uuidString: "01a0fae1-503b-722e-b8a7-10e6c36009ab")!
  private static let settlementID = UUID(uuidString: "01a0fae1-519c-7237-b3e2-2de53a4277df")!
  private static let groupPath = "/v1/groups/01a0fae1-4f3a-7c92-8675-9d81e1b3ac9e"

  private let input = SettlementInput(
    fromMemberID: grandma, toMemberID: me, amount: Money(minorUnits: 500, currency: "INR"), settledOn: "2026-10-02",
    note: "Cash")

  @Test func recordSendsTheInputAndMapsTheSettlement() async throws {
    let transport = try PathTransport([
      "POST \(Self.groupPath)/settlements": .fixture("settlement-record-201", status: .created)
    ])
    let repository = try makeRepository(transport)

    let s = try await repository.record(
      groupID: Self.groupID, input: input, acknowledgeWarnings: false, key: WriteKey())

    #expect(s.id == Self.settlementID)
    #expect(s.fromMemberID == Self.grandma && s.toMemberID == Self.me)
    #expect(s.amount == Money(minorUnits: 500, currency: "INR"))
    #expect(s.state == .accepted && s.version == 1 && s.note == "Cash")
    let sent = try #require(await transport.sent.first)
    let body = try #require(try JSONSerialization.jsonObject(with: sent.body) as? [String: Any])
    #expect(body["acknowledge_warnings"] == nil)
    #expect(body["settled_on"] as? String == "2026-10-02")
  }

  @Test func anOverpaymentNeedsConfirmation() async throws {
    let transport = try PathTransport([
      "POST \(Self.groupPath)/settlements": .fixture(
        "settlement-422-confirmation", status: .unprocessableContent, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.needsConfirmation(["overpayment"])) {
      try await repository.record(groupID: Self.groupID, input: input, acknowledgeWarnings: false, key: WriteKey())
    }
  }

  @Test func acknowledgingSendsTheFlag() async throws {
    let transport = try PathTransport([
      "POST \(Self.groupPath)/settlements": .fixture("settlement-record-201", status: .created)
    ])
    let repository = try makeRepository(transport)

    _ = try await repository.record(groupID: Self.groupID, input: input, acknowledgeWarnings: true, key: WriteKey())

    let sent = try #require(await transport.sent.first)
    let body = try #require(try JSONSerialization.jsonObject(with: sent.body) as? [String: Any])
    #expect(body["acknowledge_warnings"] as? Bool == true)
  }

  @Test func listMapsThePage() async throws {
    let transport = try PathTransport(["GET \(Self.groupPath)/settlements?limit=50": .fixture("settlements-page")])
    let repository = try makeRepository(transport)

    let page = try await repository.settlements(groupID: Self.groupID, cursor: nil)

    #expect(page.items.map(\.id) == [Self.settlementID])
  }

  @Test func withdrawSendsTheVersion() async throws {
    let transport = try PathTransport([
      "POST /v1/settlements/\(Self.settlementID.uuidString.lowercased())/withdraw": .fixture("settlement-withdraw-200")
    ])
    let repository = try makeRepository(transport)

    let s = try await repository.withdraw(id: Self.settlementID, version: 1, key: WriteKey())

    #expect(s.state == .withdrawn && s.version == 2)
  }

  @Test func withdrawingTwiceIsInvalidState() async throws {
    let transport = try PathTransport([
      "POST /v1/settlements/\(Self.settlementID.uuidString.lowercased())/withdraw": .fixture(
        "settlement-withdraw-409", status: .conflict, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.problem(.invalidState)) {
      try await repository.withdraw(id: Self.settlementID, version: 2, key: WriteKey())
    }
  }

  private func makeRepository(_ transport: PathTransport) throws -> SettlementsAPIRepository {
    SettlementsAPIRepository(
      api: APISession(
        serverURL: try #require(URL(string: "http://localhost:8080")), transport: transport,
        tokens: InMemoryTokenStore(.sample)))
  }
}
