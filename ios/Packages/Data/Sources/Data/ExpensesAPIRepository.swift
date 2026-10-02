import APIClient
import Domain
import Foundation
import OpenAPIRuntime

/// Expenses over the generated client (FR-E*). Shares always come from the
/// server (ADR-0006).
public struct ExpensesAPIRepository: ExpensesRepository {
  private let api: APISession
  private var client: Client { api.client }

  public init(api: APISession) {
    self.api = api
  }

  public func preview(groupID: UUID, input: ExpenseInput) async throws(ServiceError) -> ExpensePreview {
    let output = try await send {
      try await client.previewExpense(
        path: .init(groupId: groupID.uuidString.lowercased()), body: .json(ExpenseMapper.input(input)))
    }
    switch output {
    case .ok(let ok):
      let p = try decoding { try ok.body.json }
      return ExpensePreview(amount: ExpenseMapper.money(p.amount), shares: try p.shares.map(ExpenseMapper.share))
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .conflict(let r): throw problem(409) { try r.body.applicationProblemJson }
    case .contentTooLarge(let r): throw problem(413) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func createExpense(groupID: UUID, input: ExpenseInput, key: WriteKey) async throws(ServiceError) -> Expense {
    let output = try await send {
      try await client.createExpense(
        path: .init(groupId: groupID.uuidString.lowercased()),
        headers: .init(idempotencyKey: key.value.uuidString),
        body: .json(ExpenseMapper.input(input)))
    }
    switch output {
    case .created(let created): return try ExpenseMapper.expense(decoding { try created.body.json })
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

  public func expenses(groupID: UUID, cursor: String?) async throws(ServiceError) -> ExpensePage {
    let output = try await send {
      try await client.listExpenses(
        path: .init(groupId: groupID.uuidString.lowercased()), query: .init(cursor: cursor, limit: 50))
    }
    switch output {
    case .ok(let ok):
      let page = try decoding { try ok.body.json }
      return ExpensePage(items: try page.items.map(ExpenseMapper.summary), nextCursor: page.nextCursor)
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }

  public func expense(id: UUID) async throws(ServiceError) -> Expense {
    let output = try await send { try await client.getExpense(path: .init(expenseId: id.uuidString.lowercased())) }
    switch output {
    case .ok(let ok): return try ExpenseMapper.expense(decoding { try ok.body.json })
    case .badRequest(let r): throw problem(400) { try r.body.applicationProblemJson }
    case .unauthorized(let r): throw problem(401) { try r.body.applicationProblemJson }
    case .forbidden(let r): throw problem(403) { try r.body.applicationProblemJson }
    case .notFound(let r): throw problem(404) { try r.body.applicationProblemJson }
    case .internalServerError: throw .unexpected(status: 500)
    case .undocumented(let status, _): throw .unexpected(status: status)
    }
  }
}

/// Generated Expense types ⇄ Domain.
enum ExpenseMapper {
  static func input(_ i: ExpenseInput) -> Components.Schemas.ExpenseInput {
    Components.Schemas.ExpenseInput(
      payerMemberId: i.payerID.uuidString.lowercased(),
      amount: .init(minor: i.amount.minorUnits, currency: i.amount.currency),
      category: Components.Schemas.Category(rawValue: i.category.rawValue) ?? .other,
      note: i.note,
      spentOn: i.spentOn,
      split: .init(
        method: Components.Schemas.SplitMethod(rawValue: i.method.rawValue) ?? .equal,
        members: i.members.map { .init(memberId: $0.memberID.uuidString.lowercased(), input: $0.input) }))
  }

  static func money(_ m: Components.Schemas.Money) -> Money {
    Money(minorUnits: m.minor, currency: m.currency)
  }

  static func share(_ s: Components.Schemas.ShareLine) throws(ServiceError) -> Share {
    Share(memberID: try uuid(s.memberId), amount: money(s.share), input: s.input)
  }

  static func expense(_ e: Components.Schemas.Expense) throws(ServiceError) -> Expense {
    Expense(
      id: try uuid(e.id), groupID: try uuid(e.groupId), payerID: try uuid(e.payerMemberId),
      createdByID: try uuid(e.createdByMemberId), amount: money(e.amount), category: category(e.category),
      note: e.note, spentOn: e.spentOn, state: try state(e.state), version: Int(e.version),
      splitMethod: try splitMethod(e.splitMethod), shares: try e.shares.map(share))
  }

  static func summary(_ e: Components.Schemas.ExpenseSummary) throws(ServiceError) -> ExpenseSummary {
    ExpenseSummary(
      id: try uuid(e.id), payerID: try uuid(e.payerMemberId), amount: money(e.amount), category: category(e.category),
      note: e.note, spentOn: e.spentOn, state: try state(e.state))
  }

  private static func category(_ c: Components.Schemas.Category) -> Domain.Category {
    Domain.Category(rawValue: c.rawValue) ?? .other
  }

  /// An unknown method would show Shares the app can't explain: refuse it.
  private static func splitMethod(_ m: Components.Schemas.SplitMethod) throws(ServiceError) -> SplitMethod {
    guard let method = SplitMethod(rawValue: m.rawValue) else { throw .unexpected(status: nil) }
    return method
  }

  /// An unknown state is a server the app doesn't understand: refuse it
  /// rather than show it as one that counts (ADR-0009).
  private static func state(_ s: Components.Schemas.ExpenseState) throws(ServiceError) -> ExpenseState {
    guard let state = ExpenseState(rawValue: s.rawValue) else { throw .unexpected(status: nil) }
    return state
  }
}
