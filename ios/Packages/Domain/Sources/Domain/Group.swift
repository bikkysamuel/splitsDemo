import Foundation

/// A named set of Members who share expenses (GLOSSARY: Group).
public struct GroupSummary: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let name: String
  /// The Group Currency, an ISO 4217 code.
  public let currency: String
  public let state: GroupState

  public init(id: UUID, name: String, currency: String, state: GroupState) {
    self.id = id
    self.name = name
    self.currency = currency
    self.state = state
  }
}

/// A Group with its Members, as the signed-in User sees it.
public struct Group: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let name: String
  public let currency: String
  public let state: GroupState
  /// Sent back when changing the Group (NFR-R4).
  public let version: Int
  public let myMemberID: UUID
  public let members: [Member]

  public init(
    id: UUID, name: String, currency: String, state: GroupState, version: Int, myMemberID: UUID, members: [Member]
  ) {
    self.id = id
    self.name = name
    self.currency = currency
    self.state = state
    self.version = version
    self.myMemberID = myMemberID
    self.members = members
  }

  /// The signed-in User's Member.
  public var myMember: Member? { members.first { $0.id == myMemberID } }

  /// Whether the signed-in User is an Admin of this Group (FR-G3).
  public var isAdmin: Bool { myMember?.role == .admin }

  public var summary: GroupSummary { GroupSummary(id: id, name: name, currency: currency, state: state) }
}

/// A Group's lifecycle state (FR-G6).
public enum GroupState: String, Sendable {
  case active, closing, closed
}

/// A person's place in one Group (GLOSSARY: Member).
public struct Member: Equatable, Hashable, Identifiable, Sendable {
  public let id: UUID
  public let displayName: String
  public let role: Role
  public let status: Status
  /// Not yet linked to a User (GLOSSARY: Placeholder Member).
  public let isPlaceholder: Bool
  public let joinSeq: Int

  public init(id: UUID, displayName: String, role: Role, status: Status, isPlaceholder: Bool, joinSeq: Int) {
    self.id = id
    self.displayName = displayName
    self.role = role
    self.status = status
    self.isPlaceholder = isPlaceholder
    self.joinSeq = joinSeq
  }

  public enum Role: String, Sendable {
    case admin, member
  }

  public enum Status: String, Sendable {
    case active, former
  }
}
