import Domain
import SwiftUI

/// The Group screen (FR-U4). A scaffold for now: the Group Currency and
/// Members; Expenses, Balances and Settlements come with later tickets.
struct GroupView: View {
  @Bindable var viewModel: GroupViewModel
  @State private var renaming = false
  @State private var newName = ""

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
        TextField(text: $newName) { Text(LocalizedStringKey(CreateGroupView.name), bundle: .module) }
        Button {
          Task { await viewModel.rename(to: newName) }
        } label: {
          Text(LocalizedStringKey(Self.save), bundle: .module)
        }
        Button(role: .cancel) {
        } label: {
          Text(LocalizedStringKey(CreateGroupView.cancel), bundle: .module)
        }
      }
      .alert(
        Text(LocalizedStringKey(Self.renameFailed), bundle: .module),
        isPresented: Binding(get: { viewModel.renameError != nil }, set: { if !$0 { viewModel.dismissRenameError() } })
      ) {
        Button {
          viewModel.dismissRenameError()
        } label: {
          Text(LocalizedStringKey(RootView.okKey), bundle: .module)
        }
      } message: {
        Text(LocalizedStringKey(viewModel.renameError ?? ""), bundle: .module)
      }
      .task { await viewModel.load() }
  }

  @ViewBuilder private var content: some View {
    switch viewModel.state {
    case .loading:
      ProgressView()
        .accessibilityLabel(Text(LocalizedStringKey(HomeView.loading), bundle: .module))
    case .failed(let key):
      ContentUnavailableView {
        Label {
          Text(LocalizedStringKey(key), bundle: .module)
        } icon: {
          Image(systemName: "exclamationmark.triangle").accessibilityHidden(true)
        }
      }
    case .loaded(let group):
      List {
        Section {
          LabeledContent {
            Text(verbatim: CurrencyPicker.label(group.currency))
          } label: {
            Text(LocalizedStringKey(CreateGroupView.currency), bundle: .module)
          }
        }
        Section {
          ForEach(group.members) { member in
            MemberRow(member: member, isMe: member.id == group.myMemberID)
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
  nonisolated static let renameFailed = "Couldn't rename"
  nonisolated static let save = "Save"
  nonisolated static let members = "Members"
  nonisolated static let allKeys = [rename, renameTitle, renameFailed, save, members] + MemberRow.allKeys
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
