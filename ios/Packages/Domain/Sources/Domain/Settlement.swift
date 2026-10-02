import Foundation

/// A record that one Member paid another back (GLOSSARY: Settlement).
public struct Settlement: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let groupID: UUID
  public let fromMemberID: UUID
  public let toMemberID: UUID
  public let amount: Money
  /// "yyyy-MM-dd"
  public let settledOn: String
  public let note: String?
  public let createdByID: UUID
  public let state: SettlementState
  public let version: Int

  public init(
    id: UUID, groupID: UUID, fromMemberID: UUID, toMemberID: UUID, amount: Money, settledOn: String, note: String?,
    createdByID: UUID, state: SettlementState, version: Int
  ) {
    self.id = id
    self.groupID = groupID
    self.fromMemberID = fromMemberID
    self.toMemberID = toMemberID
    self.amount = amount
    self.settledOn = settledOn
    self.note = note
    self.createdByID = createdByID
    self.state = state
    self.version = version
  }
}

/// Where a Settlement is in its life (doc 06). In M1 it counts at once.
public enum SettlementState: String, Sendable {
  case pending, accepted, disputed
  case withdrawalPending = "withdrawal_pending"
  case withdrawn
}

/// A Settlement as entered (FR-S1).
public struct SettlementInput: Equatable, Hashable, Sendable {
  public var fromMemberID: UUID
  public var toMemberID: UUID
  public var amount: Money
  public var settledOn: String
  public var note: String?

  public init(fromMemberID: UUID, toMemberID: UUID, amount: Money, settledOn: String, note: String?) {
    self.fromMemberID = fromMemberID
    self.toMemberID = toMemberID
    self.amount = amount
    self.settledOn = settledOn
    self.note = note
  }
}

public struct SettlementPage: Equatable, Sendable {
  public let items: [Settlement]
  public let nextCursor: String?

  public init(items: [Settlement], nextCursor: String?) {
    self.items = items
    self.nextCursor = nextCursor
  }
}

/// Settlements (FR-S*). An overpayment answers
/// `.needsConfirmation(["overpayment"])`; record again with
/// `acknowledgeWarnings: true` to save it anyway (FR-S2, D16).
public protocol SettlementsRepository: Sendable {
  func record(
    groupID: UUID, input: SettlementInput, acknowledgeWarnings: Bool, key: WriteKey
  )
    async throws(ServiceError) -> Settlement
  func settlements(groupID: UUID, cursor: String?) async throws(ServiceError) -> SettlementPage
  func settlement(id: UUID) async throws(ServiceError) -> Settlement
  /// Withdraws a Settlement (its creator only).
  func withdraw(id: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Settlement
}
