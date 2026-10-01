import Domain
import Observation

/// Whether the Splits server answers, for the launch screen's status line.
@MainActor
@Observable
public final class ServerStatusViewModel {
  public enum Status: Equatable, Sendable {
    case unknown
    case checking
    case reachable
    case unreachable
  }

  public private(set) var status: Status = .unknown

  private let repository: any ServerHealthRepository

  public init(repository: any ServerHealthRepository) {
    self.repository = repository
  }

  /// Checks the server and updates `status`.
  public func refresh() async {
    status = .checking
    do {
      try await repository.checkHealth()
      status = .reachable
    } catch {
      status = .unreachable
    }
  }
}
