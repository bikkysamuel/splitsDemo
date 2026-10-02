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
  /// The Group's Expenses loaded so far, newest first.
  private(set) var expenses: [ExpenseSummary] = []
  private var nextCursor: String?
  private(set) var expensesError: String?
  private(set) var balances: GroupBalances?
  private(set) var balancesError: String?
  /// The latest Settlements, newest first.
  private(set) var settlements: [Settlement] = []
  private(set) var settlementsError: String?
  /// A failed change's message key (rename, grant Admin).
  private(set) var changeError: String?
  public private(set) var isRenaming = false

  let repository: any GroupsRepository
  let expensesRepository: any ExpensesRepository
  let balancesRepository: any BalancesRepository
  let settlementsRepository: any SettlementsRepository
  private var renameKeys = WriteKeys<[String]>()
  private var adminKeys = WriteKeys<[String]>()

  public init(
    groupID: UUID, repository: any GroupsRepository, expenses: any ExpensesRepository,
    balances: any BalancesRepository, settlements: any SettlementsRepository
  ) {
    self.groupID = groupID
    self.repository = repository
    self.expensesRepository = expenses
    self.balancesRepository = balances
    self.settlementsRepository = settlements
  }

  /// Loads the Group and the first page of its Expenses.
  public func load() async {
    if state.value == nil { state = .loading }
    do {
      state = .loaded(try await repository.group(id: groupID))
    } catch {
      state = .failed(ServiceErrorMessage.key(for: error))
      return
    }
    await loadExpenses(after: nil)
    await loadSettlements()
    await loadBalances()
  }

  /// The latest 50 Settlements (most Groups have far fewer; paging comes
  /// with the Expense filters, #21).
  func loadSettlements() async {
    do {
      settlements = try await settlementsRepository.settlements(groupID: groupID, cursor: nil).items
      settlementsError = nil
    } catch {
      settlementsError = ServiceErrorMessage.key(for: error)
    }
  }

  /// Reloads after a Settlement was recorded or withdrawn.
  func settlementsChanged() async {
    await loadSettlements()
    await loadBalances()
  }

  /// Balances are derived on read; reload them after anything changes.
  func loadBalances() async {
    do {
      balances = try await balancesRepository.balances(groupID: groupID)
      balancesError = nil
    } catch {
      balancesError = ServiceErrorMessage.key(for: error)
    }
  }

  var hasMoreExpenses: Bool { nextCursor != nil }

  /// Appends the next page of Expenses.
  public func loadMoreExpenses() async {
    guard let nextCursor else { return }
    await loadExpenses(after: nextCursor)
  }

  private func loadExpenses(after cursor: String?) async {
    do {
      let page = try await expensesRepository.expenses(groupID: groupID, cursor: cursor)
      expenses = cursor == nil ? page.items : expenses + page.items
      nextCursor = page.nextCursor
      expensesError = nil
    } catch {
      expensesError = ServiceErrorMessage.key(for: error)
    }
  }

  /// Reloads after an Expense was recorded.
  func expenseAdded() async {
    await loadExpenses(after: nil)
    await loadBalances()
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
      changeError = nil
    } catch {
      changeError = ServiceErrorMessage.key(for: error)
      if error == .problem(.versionConflict) { await load() }
    }
  }

  func dismissChangeError() { changeError = nil }

  /// Reloads after a Member was added, so the list shows the server's view.
  func memberAdded() async {
    await load()
  }

  /// Whether the signed-in User may make `member` an Admin.
  func canMakeAdmin(_ member: Member) -> Bool { state.value?.isAdmin == true && member.canBecomeAdmin }

  /// Makes a Member an Admin (Admins only, FR-G3), then reloads.
  public func makeAdmin(_ member: Member) async {
    guard let group = state.value else { return }
    let key = adminKeys.key(for: [member.id.uuidString, String(member.version)])
    do {
      _ = try await repository.makeAdmin(groupID: group.id, memberID: member.id, version: member.version, key: key)
      adminKeys.succeeded()
      changeError = nil
    } catch {
      changeError = ServiceErrorMessage.key(for: error)
    }
    await load()
  }
}
