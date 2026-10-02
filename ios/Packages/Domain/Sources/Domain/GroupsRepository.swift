import Foundation

/// The signed-in User's Groups (FR-G*). The server decides who may see and
/// change what: a Group the User isn't in is `.problem(.notFound)`.
///
/// Every write takes a `WriteKey`: the Idempotency-Key of one User action
/// (NFR-R1). Pass the same key when retrying that action, so the server
/// replays instead of acting twice.
public protocol GroupsRepository: Sendable {
  /// Every Group the User is an active Member of, oldest first.
  func groups() async throws(ServiceError) -> [GroupSummary]
  /// Creates a Group with the User as its first Admin, named `displayName`
  /// in it.
  func createGroup(
    name: String, currency: String, displayName: String, key: WriteKey
  ) async throws(ServiceError)
    -> Group
  func group(id: UUID) async throws(ServiceError) -> Group
  /// Renames a Group (Admins only). `version` is the one last read.
  func renameGroup(id: UUID, name: String, version: Int, key: WriteKey) async throws(ServiceError) -> Group
  /// Adds a person (FR-M1, FR-M2): with the email of a verified User they
  /// join at once; otherwise, or without an email, a Placeholder Member.
  func addMember(
    groupID: UUID, displayName: String, email: String?, key: WriteKey
  ) async throws(ServiceError)
    -> Member
  /// Makes a Member an Admin (Admins only, FR-G3).
  func makeAdmin(groupID: UUID, memberID: UUID, version: Int, key: WriteKey) async throws(ServiceError) -> Member
}

/// Non-financial UI preferences kept on the device (ADR-0005, FR-U5).
public protocol PreferencesRepository: Sendable {
  /// The Group Currency that pre-fills Create Group.
  func defaultCurrency() -> String
  func setDefaultCurrency(_ code: String)
  /// The Group the Report tab showed last (Q81), if any.
  func lastReportGroupID() -> UUID?
  func setLastReportGroupID(_ id: UUID)
}
