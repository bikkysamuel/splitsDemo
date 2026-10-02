import Domain
import Foundation

/// A scripted `GroupsRepository` for ViewModel tests.
actor FakeGroupsRepository: GroupsRepository {
  var groupsResult: Result<[GroupSummary], ServiceError> = .success([])
  var createResult: Result<Group, ServiceError> = .success(.trip)
  var groupResult: Result<Group, ServiceError> = .success(.trip)
  var renameResults: [Result<Group, ServiceError>] = []
  private(set) var created: [(name: String, currency: String, displayName: String)] = []
  private(set) var keys: [WriteKey] = []
  private(set) var renames: [(name: String, version: Int)] = []

  func set(groups: Result<[GroupSummary], ServiceError>) { groupsResult = groups }
  func set(create: Result<Group, ServiceError>) { createResult = create }
  func set(group: Result<Group, ServiceError>) { groupResult = group }
  func set(renames: [Result<Group, ServiceError>]) { renameResults = renames }

  func groups() async throws(ServiceError) -> [GroupSummary] { try groupsResult.get() }

  func createGroup(
    name: String, currency: String, displayName: String, key: WriteKey
  ) async throws(ServiceError)
    -> Group
  {
    created.append((name, currency, displayName))
    keys.append(key)
    return try createResult.get()
  }

  func group(id: UUID) async throws(ServiceError) -> Group { try groupResult.get() }

  var addResult: Result<Member, ServiceError> = .success(.bob)
  var adminResult: Result<Member, ServiceError> = .success(.bob)
  private(set) var added: [(displayName: String, email: String?)] = []
  private(set) var adminGrants: [(memberID: UUID, version: Int)] = []
  func set(add: Result<Member, ServiceError>) { addResult = add }
  func set(admin: Result<Member, ServiceError>) { adminResult = admin }

  func addMember(
    groupID: UUID, displayName: String, email: String?, key: WriteKey
  ) async throws(ServiceError)
    -> Member
  {
    added.append((displayName, email))
    keys.append(key)
    return try addResult.get()
  }

  func makeAdmin(groupID: UUID, memberID: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Member {
    adminGrants.append((memberID, version))
    return try adminResult.get()
  }

  func renameGroup(id: UUID, name: String, version: Int, key: WriteKey) async throws(ServiceError) -> Group {
    renames.append((name, version))
    keys.append(key)
    return try renameResults.isEmpty ? groupResult.get() : renameResults.removeFirst().get()
  }
}

final class FakePreferences: PreferencesRepository, @unchecked Sendable {
  var currency: String
  init(currency: String = "INR") { self.currency = currency }
  func defaultCurrency() -> String { currency }
  func setDefaultCurrency(_ code: String) { currency = code }
}

extension Member {
  static let bob = Member(
    id: UUID(), displayName: "Bob", role: .member, status: .active, isPlaceholder: false, joinSeq: 2, version: 3)
  static let grandma = Member(
    id: UUID(), displayName: "Grandma", role: .member, status: .active, isPlaceholder: true, joinSeq: 3)
}

extension Group {
  static let me = UUID()
  static let trip = Group(
    id: UUID(), name: "Goa trip", currency: "INR", state: .active, version: 1, myMemberID: me,
    members: [Member(id: me, displayName: "Alice", role: .admin, status: .active, isPlaceholder: false, joinSeq: 1)])

  func with(name: String, version: Int, role: Member.Role = .admin) -> Group {
    Group(
      id: id, name: name, currency: currency, state: state, version: version, myMemberID: myMemberID,
      members: members.map {
        Member(
          id: $0.id, displayName: $0.displayName, role: $0.id == myMemberID ? role : $0.role, status: $0.status,
          isPlaceholder: $0.isPlaceholder, joinSeq: $0.joinSeq)
      })
  }
}
