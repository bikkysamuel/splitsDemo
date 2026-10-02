import Domain
import Foundation
import Observation

/// The Group screen (FR-U4). For now: name, Group Currency and Members;
/// Expenses, Balances and Settlements arrive with later tickets.
@MainActor
@Observable
public final class GroupViewModel {
  public let groupID: UUID
  private(set) var state: LoadState<Domain.Group> = .loading
  /// A failed change's message key (rename, grant Admin).
  private(set) var renameError: String?
  public private(set) var isRenaming = false

  let repository: any GroupsRepository
  private var renameKeys = WriteKeys<[String]>()
  private var adminKeys = WriteKeys<[String]>()

  public init(groupID: UUID, repository: any GroupsRepository) {
    self.groupID = groupID
    self.repository = repository
  }

  public func load() async {
    if state.value == nil { state = .loading }
    do {
      state = .loaded(try await repository.group(id: groupID))
    } catch {
      state = .failed(ServiceErrorMessage.key(for: error))
    }
  }

  var canRename: Bool { state.value?.isAdmin == true }

  /// Renames the Group (Admins only). On a version conflict the Group is
  /// reloaded, so the next try uses the current version.
  public func rename(to name: String) async {
    guard let group = state.value, !isRenaming else { return }
    isRenaming = true
    defer { isRenaming = false }
    let key = renameKeys.key(for: [name, String(group.version)])
    do {
      state = .loaded(try await repository.renameGroup(id: group.id, name: name, version: group.version, key: key))
      renameKeys.succeeded()
      renameError = nil
    } catch {
      renameError = ServiceErrorMessage.key(for: error)
      if error == .problem(.versionConflict) { await load() }
    }
  }

  func dismissRenameError() { renameError = nil }

  /// Shows a Member just added, then reloads for the server's view.
  func didAdd(_ member: Member) async {
    await load()
  }

  /// Whether the signed-in User may make `member` an Admin.
  func canMakeAdmin(_ member: Member) -> Bool { canRename && member.canBecomeAdmin }

  /// Makes a Member an Admin (Admins only, FR-G3), then reloads.
  public func makeAdmin(_ member: Member) async {
    guard let group = state.value else { return }
    let key = adminKeys.key(for: [member.id.uuidString, String(member.version)])
    do {
      _ = try await repository.makeAdmin(groupID: group.id, memberID: member.id, version: member.version, key: key)
      adminKeys.succeeded()
      renameError = nil
    } catch {
      renameError = ServiceErrorMessage.key(for: error)
    }
    await load()
  }
}
