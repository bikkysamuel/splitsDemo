import Domain
import SwiftUI

/// The Report tab: pick a Group, see its Balances and how to settle up.
struct ReportView: View {
  @Bindable var viewModel: ReportViewModel

  var body: some View {
    content
      .navigationTitle(Text(LocalizedStringKey(MainTabView.report), bundle: .module))
      .task {
        viewModel.observesSelection = true
        await viewModel.load()
      }
      .onChange(of: viewModel.selectedGroupID) {
        Task { await viewModel.loadReport() }
      }
  }

  @ViewBuilder private var content: some View {
    switch viewModel.groups {
    case .loading:
      LoadingView()
    case .failed(let key):
      FailedView(messageKey: key) { await viewModel.load() }
    case .loaded(let groups) where groups.isEmpty:
      ContentUnavailableView {
        Label {
          Text(LocalizedStringKey(MainTabView.reportEmpty), bundle: .module)
        } icon: {
          Image(systemName: "chart.bar").accessibilityHidden(true)
        }
      } description: {
        Text(LocalizedStringKey(MainTabView.reportEmptyDetail), bundle: .module)
      }
    case .loaded(let groups):
      List {
        Section {
          Picker(selection: $viewModel.selectedGroupID) {
            ForEach(groups) { g in Text(verbatim: g.name).tag(Optional(g.id)) }
          } label: {
            Text(LocalizedStringKey(Self.group), bundle: .module)
          }
        }
        switch viewModel.report {
        case .loaded(let r):
          BalancesSection(group: r.group, balances: r.balances)
        case .failed(let key):
          Section { FieldErrorText(key: key) }
        case .loading, nil:
          Section { LoadingView() }
        }
      }
      .refreshable { await viewModel.loadReport() }
    }
  }

  nonisolated static let group = "Group"
  nonisolated static let allKeys = [group]
}
