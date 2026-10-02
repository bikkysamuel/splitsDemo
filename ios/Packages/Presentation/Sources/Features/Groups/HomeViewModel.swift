import Domain
import Observation

/// Home's Groups list (FR-U4). Balances per Group come with the Home
/// summary (#24).
@MainActor
@Observable
public final class HomeViewModel {
  private(set) var state: LoadState<[GroupSummary]> = .loading

  private let repository: any GroupsRepository

  public init(repository: any GroupsRepository) {
    self.repository = repository
  }

  /// Loads, or reloads on pull-to-refresh, keeping the list on a failed
  /// reload.
  public func load() async {
    if state.value == nil { state = .loading }
    do {
      state = .loaded(try await repository.groups())
    } catch {
      if state.value == nil { state = .failed(ServiceErrorMessage.key(for: error)) }
    }
  }

  /// Adds a Group the User just created, without a reload.
  func didCreate(_ group: Domain.Group) {
    var groups = state.value ?? []
    groups.removeAll { $0.id == group.id }
    groups.append(group.summary)
    state = .loaded(groups)
  }
}
