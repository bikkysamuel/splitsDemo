import Domain
import SwiftUI

/// The signed-in shell (FR-U4): Home, Report and Settings, each with its
/// own navigation stack. The tabs fill in as later tickets land.
struct MainTabView: View {
  let user: User
  let session: AppSession
  let dependencies: AppDependencies
  @State private var home: HomeViewModel
  @State private var settings: SettingsViewModel
  @State private var report: ReportViewModel

  init(user: User, session: AppSession, dependencies: AppDependencies) {
    self.user = user
    self.session = session
    self.dependencies = dependencies
    _home = State(initialValue: HomeViewModel(repository: dependencies.groups))
    _settings = State(initialValue: SettingsViewModel(preferences: dependencies.preferences))
    _report = State(
      initialValue: ReportViewModel(
        groups: dependencies.groups, balances: dependencies.balances, preferences: dependencies.preferences))
  }

  var body: some View {
    TabView {
      Tab {
        HomeView(viewModel: home, dependencies: dependencies)
      } label: {
        Label {
          Text(LocalizedStringKey(Self.home), bundle: .module)
        } icon: {
          Image(systemName: "house")
        }
      }
      Tab {
        NavigationStack {
          ReportView(viewModel: report)
        }
      } label: {
        Label {
          Text(LocalizedStringKey(Self.report), bundle: .module)
        } icon: {
          Image(systemName: "chart.bar")
        }
      }
      Tab {
        NavigationStack {
          SettingsView(user: user, session: session, viewModel: settings)
        }
      } label: {
        Label {
          Text(LocalizedStringKey(Self.settings), bundle: .module)
        } icon: {
          Image(systemName: "gearshape")
        }
      }
    }
  }

  nonisolated static let home = "Home"
  nonisolated static let homeEmpty = "No Groups yet"
  nonisolated static let homeEmptyDetail = "Groups you create or join will show here."
  nonisolated static let report = "Report"
  nonisolated static let reportEmpty = "Nothing to report yet"
  nonisolated static let reportEmptyDetail = "Reports appear once you have Groups."
  nonisolated static let settings = "Settings"

  nonisolated static let allKeys = [
    home, homeEmpty, homeEmptyDetail, report, reportEmpty, reportEmptyDetail, settings,
  ]
}
