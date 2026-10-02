import Domain
import Foundation
import Observation

/// The Report tab (Q81): a Group picker, remembering the last Group, and
/// that Group's Balances. The Category chart comes with M3.
@MainActor
@Observable
public final class ReportViewModel {
  private(set) var groups: LoadState<[GroupSummary]> = .loading
  /// The picked Group, saved for next time.
  public var selectedGroupID: UUID? {
    didSet {
      if let selectedGroupID, selectedGroupID != oldValue { preferences.setLastReportGroupID(selectedGroupID) }
    }
  }
  private(set) var report: LoadState<Report>?

  struct Report: Equatable {
    let group: Domain.Group
    let balances: GroupBalances
  }

  private let groupsRepository: any GroupsRepository
  private let balancesRepository: any BalancesRepository
  private let preferences: any PreferencesRepository

  public init(
    groups: any GroupsRepository, balances: any BalancesRepository, preferences: any PreferencesRepository
  ) {
    self.groupsRepository = groups
    self.balancesRepository = balances
    self.preferences = preferences
  }

  /// Loads the Groups, picks the remembered one (else the first) and loads
  /// its report.
  public func load() async {
    do {
      let list = try await groupsRepository.groups()
      groups = .loaded(list)
      let remembered = preferences.lastReportGroupID()
      if selectedGroupID == nil || !list.contains(where: { $0.id == selectedGroupID }) {
        selectedGroupID = list.first(where: { $0.id == remembered })?.id ?? list.first?.id
      }
      await loadReport()
    } catch {
      groups = .failed(ServiceErrorMessage.key(for: error))
    }
  }

  /// Loads the selected Group's Balances.
  public func loadReport() async {
    guard let id = selectedGroupID else {
      report = nil
      return
    }
    if report?.value?.group.id != id { report = .loading }
    do {
      async let group = groupsRepository.group(id: id)
      async let balances = balancesRepository.balances(groupID: id)
      let r = Report(group: try await group, balances: try await balances)
      if selectedGroupID == id { report = .loaded(r) }
    } catch {
      if selectedGroupID == id {
        report = .failed(ServiceErrorMessage.key(for: error as? ServiceError ?? .unexpected(status: nil)))
      }
    }
  }
}
