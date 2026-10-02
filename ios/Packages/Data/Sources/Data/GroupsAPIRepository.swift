import APIClient
import Domain
import Foundation
import OpenAPIRuntime

/// Groups over the generated client (FR-G*). Generated types stay in Data
/// (ADR-0015).
public struct GroupsAPIRepository: GroupsRepository {
  private let api: APISession
  private var client: Client { api.client }

  public init(api: APISession) {
    self.api = api
  }

  public func groups() async throws(ServiceError) -> [GroupSummary] {
    var all: [GroupSummary] = []
    var cursor: String?
    repeat {
      let output = try await send { [cursor] in
        try await client.listGroups(query: .init(cursor: cursor, limit: 200))
      }
      switch output {
      case .ok(let ok):
        let page = try decoding { try ok.body.json }
        all += try page.items.map(GroupMapper.summary)
        cursor = page.nextCursor
      case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
      case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
      case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
      case .internalServerError: throw .unexpected(status: 500)
      case .undocumented(let status, _): throw .unexpected(status: status)
      }
    } while cursor != nil
    return all
  }

  public func createGroup(name: String, currency: String, displayName: String) async throws(ServiceError) -> Group {
    let output = try await send {
      try await client.createGroup(
        headers: .init(idempotencyKey: UUID().uuidString),
        body: .json(.init(name: name, currency: currency, displayName: displayName)))
    }
    switch output {
    case .created(let created): return try GroupMapper.group(decoding { try created.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .conflict(let r): throw problem(409) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .unprocessableContent(let r): throw problem(422) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func group(id: UUID) async throws(ServiceError) -> Group {
    let output = try await send { try await client.getGroup(path: .init(groupId: id.uuidString.lowercased())) }
    switch output {
    case .ok(let ok): return try GroupMapper.group(decoding { try ok.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func renameGroup(id: UUID, name: String, version: Int) async throws(ServiceError) -> Group {
    let output = try await send {
      try await client.renameGroup(
        path: .init(groupId: id.uuidString.lowercased()),
        headers: .init(idempotencyKey: UUID().uuidString),
        body: .json(.init(name: name, version: Int32(clamping: version))))
    }
    switch output {
    case .ok(let ok): return try GroupMapper.group(decoding { try ok.body.json })
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

/// Generated Group types → Domain.
enum GroupMapper {
  static func summary(_ api: Components.Schemas.GroupSummary) throws(ServiceError) -> GroupSummary {
    GroupSummary(id: try uuid(api.id), name: api.name, currency: api.currency, state: state(api.state))
  }

  static func group(_ api: Components.Schemas.Group) throws(ServiceError) -> Group {
    var members: [Member] = []
    for m in api.members {
      members.append(
        Member(
          id: try uuid(m.id), displayName: m.displayName,
          role: m.role == .admin ? .admin : .member,
          status: m.status == .active ? .active : .former,
          isPlaceholder: m.placeholder, joinSeq: Int(m.joinSeq)))
    }
    return Group(
      id: try uuid(api.id), name: api.name, currency: api.currency, state: state(api.state),
      version: Int(api.version), myMemberID: try uuid(api.myMemberId), members: members)
  }

  private static func state(_ s: Components.Schemas.GroupState) -> GroupState {
    switch s {
    case .active: .active
    case .closing: .closing
    case .closed: .closed
    }
  }
}
