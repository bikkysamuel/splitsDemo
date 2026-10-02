import Domain
import Foundation
import Observation

/// A Settlement, with Withdraw for its creator (FR-S3).
@MainActor
@Observable
public final class SettlementDetailViewModel {
  public let settlementID: UUID
  public let group: Domain.Group
  private(set) var state: LoadState<Settlement> = .loading
  private(set) var actionError: String?
  public private(set) var isWithdrawing = false

  private let repository: any SettlementsRepository
  private var keys = WriteKeys<Int>()

  public init(settlementID: UUID, group: Domain.Group, repository: any SettlementsRepository) {
    self.settlementID = settlementID
    self.group = group
    self.repository = repository
  }

  public func load() async {
    do {
      state = .loaded(try await repository.settlement(id: settlementID))
    } catch {
      state = .failed(ServiceErrorMessage.key(for: error))
    }
  }

  var canWithdraw: Bool {
    guard let s = state.value else { return false }
    return s.createdByID == group.myMemberID && s.state != .withdrawn
  }

  /// Withdraws the Settlement; returns true on success.
  public func withdraw() async -> Bool {
    guard let s = state.value, canWithdraw, !isWithdrawing else { return false }
    isWithdrawing = true
    defer { isWithdrawing = false }
    do {
      state = .loaded(try await repository.withdraw(id: s.id, version: s.version, key: keys.key(for: s.version)))
      keys.succeeded()
      actionError = nil
      return true
    } catch {
      actionError = ServiceErrorMessage.key(for: error)
      await load()
      return false
    }
  }

  func dismissError() { actionError = nil }

  func name(_ id: UUID) -> String? { group.member(id)?.displayName }
}
