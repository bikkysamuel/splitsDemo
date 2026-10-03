import Domain
import SwiftUI

/// The Group screen (FR-U4). A scaffold for now: the Group Currency and
/// Members; Expenses, Balances and Settlements come with later tickets.
struct GroupView: View {
  @Bindable var viewModel: GroupViewModel
  @State private var renaming = false
  @State private var newName = ""
  @State private var addMember: AddMemberViewModel?
  @State private var addExpense: AddExpenseViewModel?
  @State private var recordSettlement: RecordSettlementViewModel?
  @State private var promoting: Member?
  @State private var filtering = false

  var body: some View {
    content
      .navigationTitle(Text(verbatim: viewModel.state.value?.name ?? ""))
      .toolbar {
        if let group = viewModel.state.value {
          // FR-U4: Add Expense is the Group screen's one primary action.
          ToolbarItem(placement: .primaryAction) {
            Button {
              addExpense = AddExpenseViewModel(group: group, repository: viewModel.expensesRepository)
            } label: {
              Label {
                Text(LocalizedStringKey(AddExpenseView.title), bundle: .module)
              } icon: {
                Image(systemName: "plus")
              }
            }
          }
        }
        if let group = viewModel.state.value, group.state != .closed {
          ToolbarItem(placement: .secondaryAction) {
            Button {
              recordSettlement = RecordSettlementViewModel(group: group, repository: viewModel.settlementsRepository)
            } label: {
              Text(LocalizedStringKey(RecordSettlementView.title), bundle: .module)
            }
          }
        }
        if viewModel.canRename {
          ToolbarItem(placement: .secondaryAction) {
            Button {
              newName = viewModel.state.value?.name ?? ""
              renaming = true
            } label: {
              Text(LocalizedStringKey(Self.rename), bundle: .module)
            }
          }
        }
      }
      .sheet(item: $recordSettlement) { model in
        NavigationStack {
          RecordSettlementView(viewModel: model) { _ in
            recordSettlement = nil
            Task { await viewModel.settlementsChanged() }
          }
        }
      }
      .sheet(isPresented: $filtering) {
        if let group = viewModel.state.value {
          NavigationStack {
            ExpenseFilterView(group: group, filter: viewModel.filter) { filter in
              filtering = false
              Task { await viewModel.apply(filter) }
            }
          }
        }
      }
      .sheet(item: $addExpense) { model in
        NavigationStack {
          AddExpenseView(viewModel: model) { _ in
            addExpense = nil
            Task { await viewModel.expenseAdded() }
          }
        }
      }
      .alert(Text(LocalizedStringKey(Self.renameTitle), bundle: .module), isPresented: $renaming) {
        TextField(text: $newName) { Text(LocalizedStringKey(GroupKeys.name), bundle: .module) }
        Button {
          Task { await viewModel.rename(to: newName) }
        } label: {
          Text(LocalizedStringKey(CommonKeys.save), bundle: .module)
        }
        Button(role: .cancel) {
        } label: {
          Text(LocalizedStringKey(CommonKeys.cancel), bundle: .module)
        }
      }
      .alert(
        Text(LocalizedStringKey(Self.renameFailed), bundle: .module),
        isPresented: Binding(get: { viewModel.changeError != nil }, set: { if !$0 { viewModel.dismissChangeError() } })
      ) {
        Button {
          viewModel.dismissChangeError()
        } label: {
          Text(LocalizedStringKey(CommonKeys.ok), bundle: .module)
        }
      } message: {
        Text(LocalizedStringKey(viewModel.changeError ?? ""), bundle: .module)
      }
      .sheet(item: $addMember) { model in
        NavigationStack {
          AddMemberView(viewModel: model) { _ in
            addMember = nil
            Task { await viewModel.memberAdded() }
          }
        }
      }
      .confirmationDialog(
        Text(LocalizedStringKey(Self.makeAdminTitle), bundle: .module),
        isPresented: Binding(get: { promoting != nil }, set: { if !$0 { promoting = nil } }),
        titleVisibility: .visible, presenting: promoting
      ) { member in
        Button {
          Task { await viewModel.makeAdmin(member) }
        } label: {
          Text(LocalizedStringKey(Self.makeAdmin), bundle: .module)
        }
      } message: { member in
        Text(verbatim: Self.makeAdminText(member.displayName))
      }
      .task { await viewModel.load() }
  }

  @ViewBuilder private var content: some View {
    switch viewModel.state {
    case .loading:
      LoadingView()
    case .failed(let key):
      FailedView(messageKey: key) { await viewModel.load() }
    case .loaded(let group):
      List {
        Section {
          LabeledContent {
            Text(verbatim: CurrencyPicker.label(group.currency))
          } label: {
            Text(LocalizedStringKey(GroupKeys.currency), bundle: .module)
          }
        }
        if let balances = viewModel.balances {
          BalancesSection(
            group: group, balances: balances,
            onSettle: group.state == .closed
              ? nil
              : { suggestion in
                recordSettlement = RecordSettlementViewModel(
                  group: group, repository: viewModel.settlementsRepository, suggestion: suggestion)
              })
        }
        if let key = viewModel.balancesError {
          Section {
            FieldErrorText(key: key)
            Button {
              Task { await viewModel.loadBalances() }
            } label: {
              Text(LocalizedStringKey(CommonKeys.tryAgain), bundle: .module).frame(minHeight: 44)
            }
          } header: {
            Text(LocalizedStringKey(BalancesSection.balancesHeader), bundle: .module)
          }
        }
        Section {
          if viewModel.expenses.isEmpty {
            Text(LocalizedStringKey(viewModel.filter.isActive ? Self.noMatches : Self.noExpenses), bundle: .module)
              .foregroundStyle(.secondary)
          }
          ForEach(viewModel.expenses) { e in
            NavigationLink {
              ExpenseDetailView(
                viewModel: ExpenseDetailViewModel(
                  expenseID: e.id, group: group, repository: viewModel.expensesRepository)
              ) {
                Task { await viewModel.expenseAdded() }
              }
            } label: {
              ExpenseRow(expense: e, payerName: group.member(e.payerID)?.displayName)
            }
            .task { await viewModel.expenseAppeared(e) }
          }
          if viewModel.hasMoreExpenses {
            ProgressView()
              .frame(maxWidth: .infinity)
              .accessibilityLabel(Text(LocalizedStringKey(CommonKeys.loading), bundle: .module))
          }
          if let key = viewModel.expensesError {
            FieldErrorText(key: key)
          }
        } header: {
          HStack {
            Text(LocalizedStringKey(Self.expenses), bundle: .module)
            Spacer()
            Button {
              filtering = true
            } label: {
              Label {
                Text(LocalizedStringKey(Self.filter), bundle: .module)
              } icon: {
                Image(
                  systemName: viewModel.filter.isActive
                    ? "line.3.horizontal.decrease.circle.fill" : "line.3.horizontal.decrease.circle")
              }
              .labelStyle(.iconOnly)
              .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityValue(
              Text(LocalizedStringKey(viewModel.filter.isActive ? Self.filterOn : Self.filterOff), bundle: .module))
          }
        }
        if let key = viewModel.settlementsError {
          Section {
            FieldErrorText(key: key)
          } header: {
            Text(LocalizedStringKey(Self.settlements), bundle: .module)
          }
        }
        if !viewModel.settlements.isEmpty {
          Section {
            ForEach(viewModel.settlements) { s in
              NavigationLink {
                SettlementDetailView(
                  viewModel: SettlementDetailViewModel(
                    settlementID: s.id, group: group, repository: viewModel.settlementsRepository)
                ) {
                  Task { await viewModel.settlementsChanged() }
                }
              } label: {
                SettlementRow(
                  settlement: s, fromName: group.member(s.fromMemberID)?.displayName,
                  toName: group.member(s.toMemberID)?.displayName)
              }
            }
          } header: {
            Text(LocalizedStringKey(Self.settlements), bundle: .module)
          }
        }
        Section {
          ForEach(group.members) { member in
            MemberRow(member: member, isMe: member.id == group.myMemberID)
              .swipeActions {
                if viewModel.canMakeAdmin(member) {
                  Button {
                    promoting = member
                  } label: {
                    Text(LocalizedStringKey(Self.makeAdmin), bundle: .module)
                  }
                  .tint(.blue)
                }
              }
              .accessibilityAction(named: Text(LocalizedStringKey(Self.makeAdmin), bundle: .module)) {
                if viewModel.canMakeAdmin(member) { promoting = member }
              }
          }
          Button {
            addMember = AddMemberViewModel(groupID: group.id, repository: viewModel.repository)
          } label: {
            Label {
              Text(LocalizedStringKey(AddMemberView.title), bundle: .module)
            } icon: {
              Image(systemName: "person.badge.plus").accessibilityHidden(true)
            }
            .frame(minHeight: 44)
          }
        } header: {
          Text(LocalizedStringKey(Self.members), bundle: .module)
        }
      }
      .refreshable { await viewModel.load() }
    }
  }

  nonisolated static let rename = "Rename"
  nonisolated static let renameTitle = "Rename Group"
  nonisolated static let renameFailed = "Couldn't make the change"
  nonisolated static let makeAdmin = "Make Admin"
  nonisolated static let makeAdminTitle = "Make this Member an Admin?"
  nonisolated static let makeAdminMessage = "%@ will be able to rename the Group, remove Members and grant Admin."

  /// The grant-Admin confirmation, from the one catalog key above.
  static func makeAdminText(_ name: String) -> String {
    String(format: String(localized: String.LocalizationValue(makeAdminMessage), bundle: .module), name)
  }
  nonisolated static let members = "Members"
  nonisolated static let expenses = "Expenses"
  nonisolated static let noExpenses = "No Expenses yet. Add the first with +."
  nonisolated static let noMatches = "No Expenses match these filters."
  nonisolated static let filter = "Filter"
  nonisolated static let filterOn = "On"
  nonisolated static let filterOff = "Off"
  nonisolated static let settlements = "Settlements"
  nonisolated static let allKeys =
    [
      rename, renameTitle, renameFailed, members, makeAdmin, makeAdminTitle, makeAdminMessage, expenses, noExpenses,
      noMatches, filter, filterOn, filterOff, settlements,
    ]
    + MemberRow.allKeys
}

/// One Member, with Admin, Placeholder and "you" badges.
struct MemberRow: View {
  let member: Member
  let isMe: Bool

  var body: some View {
    HStack {
      Text(verbatim: member.displayName)
      if isMe {
        Text(LocalizedStringKey(Self.you), bundle: .module).foregroundStyle(.secondary)
      }
      Spacer()
      if member.role == .admin {
        Text(LocalizedStringKey(Self.admin), bundle: .module).font(.footnote).foregroundStyle(.secondary)
      }
      if member.isPlaceholder {
        Text(LocalizedStringKey(Self.placeholder), bundle: .module).font(.footnote).foregroundStyle(.secondary)
      }
    }
    .accessibilityElement(children: .combine)
  }

  nonisolated static let you = "(you)"
  nonisolated static let admin = "Admin"
  nonisolated static let placeholder = "Placeholder"
  nonisolated static let allKeys = [you, admin, placeholder]
}
