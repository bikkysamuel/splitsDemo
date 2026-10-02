import APIClient
import Domain
import Foundation
import OpenAPIRuntime

/// Balances over the generated client: computed by the server, never here
/// (ADR-0006).
public struct BalancesAPIRepository: BalancesRepository {
  private let api: APISession
  private var client: Client { api.client }

  public init(api: APISession) {
    self.api = api
  }

  public func balances(groupID: UUID) async throws(ServiceError) -> GroupBalances {
    let output = try await send { try await client.getBalances(path: .init(groupId: groupID.uuidString.lowercased())) }
    switch output {
    case .ok(let ok):
      let b = try decoding { try ok.body.json }
      var balances: [MemberBalance] = []
      for x in b.balances {
        balances.append(MemberBalance(memberID: try uuid(x.memberId), balance: ExpenseMapper.money(x.balance)))
      }
      var suggestions: [SettleUpSuggestion] = []
      for s in b.suggestions {
        suggestions.append(
          SettleUpSuggestion(
            fromMemberID: try uuid(s.fromMemberId), toMemberID: try uuid(s.toMemberId),
            amount: ExpenseMapper.money(s.amount)))
      }
      return GroupBalances(balances: balances, suggestions: suggestions)
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }
}
