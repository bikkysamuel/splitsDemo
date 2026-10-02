import Domain
import Foundation

/// A scripted `GroupsRepository` for ViewModel tests.
actor FakeGroupsRepository: GroupsRepository {
  var groupsResult: Result<[GroupSummary], ServiceError> = .success([])
  var createResult: Result<Group, ServiceError> = .success(.trip)
  var groupResult: Result<Group, ServiceError> = .success(.trip)
  var renameResults: [Result<Group, ServiceError>] = []
  private(set) var created: [(name: String, currency: String, displayName: String)] = []
  private(set) var renames: [(name: String, version: Int)] = []

  func set(groups: Result<[GroupSummary], ServiceError>) { groupsResult = groups }
  func set(create: Result<Group, ServiceError>) { createResult = create }
  func set(group: Result<Group, ServiceError>) { groupResult = group }
  func set(renames: [Result<Group, ServiceError>]) { renameResults = renames }

  func groups() async throws(ServiceError) -> [GroupSummary] { try groupsResult.get() }

  func createGroup(name: String, currency: String, displayName: String) async throws(ServiceError) -> Group {
    created.append((name, currency, displayName))
    return try createResult.get()
  }

  func group(id: UUID) async throws(ServiceError) -> Group { try groupResult.get() }

  func renameGroup(id: UUID, name: String, version: Int) async throws(ServiceError) -> Group {
    renames.append((name, version))
    return try renameResults.isEmpty ? groupResult.get() : renameResults.removeFirst().get()
  }
}

final class FakePreferences: PreferencesRepository, @unchecked Sendable {
  var currency: String
  init(currency: String = "INR") { self.currency = currency }
  func defaultCurrency() -> String { currency }
  func setDefaultCurrency(_ code: String) { currency = code }
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
