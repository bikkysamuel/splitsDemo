import APIClient
import Data
import Domain
import Features
import SwiftUI

/// The composition root: the only module that imports every layer
/// (ADR-0015). It builds the infrastructure, wraps it in repositories and
/// hands those to the ViewModels.
@main
struct SplitsApp: App {
  @State private var serverStatus: ServerStatusViewModel

  init() {
    _serverStatus = State(
      initialValue: ServerStatusViewModel(repository: ServerHealthAPIRepository(configuration: Self.apiConfiguration()))
    )
  }

  var body: some Scene {
    WindowGroup {
      ServerStatusView(viewModel: serverStatus)
    }
  }

  /// Reads the base URL from the build configuration. A missing or insecure
  /// URL is a build misconfiguration, so the app stops at launch.
  private static func apiConfiguration() -> APIConfiguration {
    #if DEBUG
      let requiresHTTPS = false
    #else
      let requiresHTTPS = true
    #endif
    do {
      return try APIConfiguration(infoDictionary: Bundle.main.infoDictionary ?? [:], requiresHTTPS: requiresHTTPS)
    } catch {
      fatalError("Invalid API configuration: \(error)")
    }
  }
}
