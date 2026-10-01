import Foundation
import Testing

@testable import Features

/// User-facing strings come only from the String Catalog (NFR): a key the
/// code uses but the catalog lacks would silently show the raw key.
struct StringCatalogTests {
  @Test(arguments: [
    ServerStatusViewModel.Status.unknown, .checking, .reachable, .unreachable,
  ])
  func everyServerStatusTitleIsInTheCatalog(status: ServerStatusViewModel.Status) {
    let key = ServerStatusAppearance(status).titleKey

    #expect(Bundle.module.localizedString(forKey: key, value: missing, table: nil) != missing)
  }

  @Test func theRetryButtonTitleIsInTheCatalog() {
    let key = ServerStatusView.retryKey

    #expect(Bundle.module.localizedString(forKey: key, value: missing, table: nil) != missing)
  }

  private let missing = "\u{0}missing"
}
