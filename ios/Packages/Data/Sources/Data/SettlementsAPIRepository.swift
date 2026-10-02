import APIClient
import Domain
import Foundation
import OpenAPIRuntime

/// Settlements over the generated client (FR-S*).
public struct SettlementsAPIRepository: SettlementsRepository {
  private let api: APISession
  private var client: Client { api.client }

  public init(api: APISession) {
    self.api = api
  }

  public func record(
    groupID: UUID, input: SettlementInput, acknowledgeWarnings: Bool, key: WriteKey
  )
    async throws(ServiceError) -> Settlement
  {
    let body = Components.Schemas.SettlementInput(
      fromMemberId: input.fromMemberID.uuidString.lowercased(), toMemberId: input.toMemberID.uuidString.lowercased(),
      amount: .init(minor: input.amount.minorUnits, currency: input.amount.currency), settledOn: input.settledOn,
      note: input.note, acknowledgeWarnings: acknowledgeWarnings ? true : nil)
    let output = try await send {
      try await client.recordSettlement(
        path: .init(groupId: groupID.uuidString.lowercased()), headers: .init(idempotencyKey: key.value.uuidString),
        body: .json(body))
    }
    switch output {
    case .created(let created): return try SettlementMapper.settlement(decoding { try created.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .conflict(let r): throw problem(409) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .unprocessableContent(let r): throw problem(422) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func settlements(groupID: UUID, cursor: String?) async throws(ServiceError) -> SettlementPage {
    let output = try await send {
      try await client.listSettlements(
        path: .init(groupId: groupID.uuidString.lowercased()), query: .init(cursor: cursor, limit: 50))
    }
    switch output {
    case .ok(let ok):
      let page = try decoding { try ok.body.json }
      return SettlementPage(items: try page.items.map(SettlementMapper.settlement), nextCursor: page.nextCursor)
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func settlement(id: UUID) async throws(ServiceError) -> Settlement {
    let output = try await send {
      try await client.getSettlement(path: .init(settlementId: id.uuidString.lowercased()))
    }
    switch output {
    case .ok(let ok): return try SettlementMapper.settlement(decoding { try ok.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func withdraw(id: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Settlement {
    let output = try await send {
      try await client.withdrawSettlement(
        path: .init(settlementId: id.uuidString.lowercased()), headers: .init(idempotencyKey: key.value.uuidString),
        body: .json(.init(version: Int32(clamping: version))))
    }
    switch output {
    case .ok(let ok): return try SettlementMapper.settlement(decoding { try ok.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .conflict(let r): throw problem(409) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .unprocessableContent(let r): throw problem(422) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }
}

enum SettlementMapper {
  static func settlement(_ s: Components.Schemas.Settlement) throws(ServiceError) -> Settlement {
    guard let state = SettlementState(rawValue: s.state.rawValue) else { throw .unexpected(status: nil) }
    return Settlement(
      id: try uuid(s.id), groupID: try uuid(s.groupId), fromMemberID: try uuid(s.fromMemberId),
      toMemberID: try uuid(s.toMemberId), amount: ExpenseMapper.money(s.amount), settledOn: s.settledOn, note: s.note,
      createdByID: try uuid(s.createdByMemberId), state: state, version: Int(s.version))
  }
}
