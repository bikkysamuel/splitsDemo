import Domain
import SwiftUI

/// The Group screen (FR-U4). A scaffold for now: the Group Currency and
/// Members; Expenses, Balances and Settlements come with later tickets.
struct GroupView: View {
  @Bindable var viewModel: GroupViewModel
  @State private var renaming = false
  @State private var newName = ""
  @State private var addMember: AddMemberViewModel?
  @State private var promoting: Member?

  var body: some View {
    content
      .navigationTitle(Text(verbatim: viewModel.state.value?.name ?? ""))
      .toolbar {
        if viewModel.canRename {
          ToolbarItem(placement: .primaryAction) {
            Button {
              newName = viewModel.state.value?.name ?? ""
              renaming = true
            } label: {
              Text(LocalizedStringKey(Self.rename), bundle: .module)
            }
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
        isPresented: Binding(get: { viewModel.renameError != nil }, set: { if !$0 { viewModel.dismissRenameError() } })
      ) {
        Button {
          viewModel.dismissRenameError()
        } label: {
          Text(LocalizedStringKey(CommonKeys.ok), bundle: .module)
        }
      } message: {
        Text(LocalizedStringKey(viewModel.renameError ?? ""), bundle: .module)
      }
      .sheet(item: $addMember) { model in
        NavigationStack {
          AddMemberView(viewModel: model) { member in
            addMember = nil
            Task { await viewModel.didAdd(member) }
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
        Text(
          LocalizedStringKey("\(member.displayName) will be able to rename the Group, remove Members and grant Admin."),
          bundle: .module)
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
  nonisolated static let members = "Members"
  nonisolated static let allKeys =
    [rename, renameTitle, renameFailed, members, makeAdmin, makeAdminTitle, makeAdminMessage] + MemberRow.allKeys
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
