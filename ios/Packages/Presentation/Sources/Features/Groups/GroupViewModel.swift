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
  /// A failed rename's message key.
  private(set) var renameError: String?
  public private(set) var isRenaming = false

  private let repository: any GroupsRepository

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

  var canRename: Bool { state.value?.iAmAdmin == true }

  /// Renames the Group (Admins only). On a version conflict the Group is
  /// reloaded, so the next try uses the current version.
  public func rename(to name: String) async {
    guard let group = state.value, !isRenaming else { return }
    isRenaming = true
    defer { isRenaming = false }
    do {
      state = .loaded(try await repository.renameGroup(id: group.id, name: name, version: group.version))
      renameError = nil
    } catch {
      renameError = ServiceErrorMessage.key(for: error)
      if error == .problem(.versionConflict) { await load() }
    }
  }

  func dismissRenameError() { renameError = nil }
}
