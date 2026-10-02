import Domain
import SwiftUI

/// Home (FR-U4): the User's Groups and Create Group. Tapping a Group opens
/// its screen.
struct HomeView: View {
  @Bindable var viewModel: HomeViewModel
  let dependencies: AppDependencies
  @State private var path: [UUID] = []
  @State private var createGroup: CreateGroupViewModel?

  var body: some View {
    NavigationStack(path: $path) {
      content
        .navigationTitle(Text(LocalizedStringKey(MainTabView.home), bundle: .module))
        .toolbar {
          ToolbarItem(placement: .primaryAction) {
            Button {
              startCreating()
            } label: {
              Label {
                Text(LocalizedStringKey(Self.createGroup), bundle: .module)
              } icon: {
                Image(systemName: "plus")
              }
            }
          }
        }
        .navigationDestination(for: UUID.self) { id in
          GroupView(viewModel: GroupViewModel(groupID: id, repository: dependencies.groups))
        }
        .sheet(item: $createGroup) { model in
          NavigationStack {
            CreateGroupView(viewModel: model) { group in
              createGroup = nil
              viewModel.didCreate(group)
              path.append(group.id)
            }
          }
        }
    }
    .task { await viewModel.load() }
  }

  @ViewBuilder private var content: some View {
    switch viewModel.state {
    case .loading:
      ProgressView()
        .accessibilityLabel(Text(LocalizedStringKey(Self.loading), bundle: .module))
    case .failed(let key):
      ContentUnavailableView {
        Label {
          Text(LocalizedStringKey(key), bundle: .module)
        } icon: {
          Image(systemName: "exclamationmark.triangle").accessibilityHidden(true)
        }
      } actions: {
        Button {
          Task { await viewModel.load() }
        } label: {
          Text(LocalizedStringKey(SplashView.retryKey), bundle: .module).frame(minHeight: 44)
        }
        .buttonStyle(.bordered)
      }
    case .loaded(let groups) where groups.isEmpty:
      ContentUnavailableView {
        Label {
          Text(LocalizedStringKey(MainTabView.homeEmpty), bundle: .module)
        } icon: {
          Image(systemName: "person.3").accessibilityHidden(true)
        }
      } description: {
        Text(LocalizedStringKey(MainTabView.homeEmptyDetail), bundle: .module)
      } actions: {
        Button {
          startCreating()
        } label: {
          Text(LocalizedStringKey(Self.createGroup), bundle: .module).frame(minHeight: 44)
        }
        .buttonStyle(.borderedProminent)
      }
    case .loaded(let groups):
      List(groups) { group in
        NavigationLink(value: group.id) {
          VStack(alignment: .leading, spacing: 2) {
            Text(verbatim: group.name).font(.body)
            Text(verbatim: group.currency).font(.footnote).foregroundStyle(.secondary)
          }
          .accessibilityElement(children: .combine)
        }
      }
      .refreshable { await viewModel.load() }
    }
  }

  private func startCreating() {
    createGroup = CreateGroupViewModel(repository: dependencies.groups, preferences: dependencies.preferences)
  }

  nonisolated static let createGroup = "Create Group"
  nonisolated static let loading = "Loading…"
  nonisolated static let allKeys = [createGroup, loading]
}

extension CreateGroupViewModel: Identifiable {}
