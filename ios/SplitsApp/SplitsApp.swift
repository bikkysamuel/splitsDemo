import APIClient
import Data
import Domain
import Features
import SwiftUI

/// The composition root: the only module that imports every layer
/// (ADR-0015). It builds the infrastructure, wraps it in repositories and
/// hands those to the app-level session, which drives the root view.
@main
struct SplitsApp: App {
  @State private var session: AppSession

  init() {
    let repository = AuthAPIRepository(configuration: Self.apiConfiguration(), tokens: KeychainTokenStore())
    _session = State(initialValue: AppSession(repository: repository))
  }

  var body: some Scene {
    WindowGroup {
      RootView(session: session)
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
