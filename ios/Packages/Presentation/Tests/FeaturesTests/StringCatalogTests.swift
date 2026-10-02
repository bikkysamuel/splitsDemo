import Domain
import Foundation
import Testing

@testable import Features

/// User-facing strings come only from the String Catalog (NFR): a key the
/// code uses but the catalog lacks would silently show the raw key.
struct StringCatalogTests {
  @Test(arguments: ServiceErrorMessage.allKeys)
  func everyErrorMessageIsInTheCatalog(key: String) {
    #expect(isInCatalog(key))
  }

  @Test(arguments: Keys.all)
  func everyAuthScreenStringIsInTheCatalog(key: String) {
    #expect(isInCatalog(key))
  }

  @Test(arguments: SettingsView.allKeys + RootView.allKeys)
  func everySettingsAndAlertStringIsInTheCatalog(key: String) {
    #expect(isInCatalog(key))
  }

  @Test(arguments: GroupKeys.all + CommonKeys.all + CreateGroupView.allKeys + GroupView.allKeys + AddMemberView.allKeys)
  func everyGroupScreenStringIsInTheCatalog(key: String) {
    #expect(isInCatalog(key))
  }

  @Test(arguments: MainTabView.allKeys)
  func everyTabStringIsInTheCatalog(key: String) {
    #expect(isInCatalog(key))
  }

  @Test(arguments: [
    SplashView.titleKey, SplashView.checkingKey, VerifyEmailViewModel.resentMessage,
  ])
  func everySplashAndVerificationStringIsInTheCatalog(key: String) {
    #expect(isInCatalog(key))
  }

  @Test func theCodeSentMessageFormatsTheEmail() {
    let format = Bundle.module.localizedString(forKey: Keys.codeSentFormat, value: nil, table: nil)

    #expect(String(format: format, "alice@example.com") == "We sent a 6-digit code to alice@example.com.")
  }

  private func isInCatalog(_ key: String) -> Bool {
    let missing = "\u{0}missing"
    return Bundle.module.localizedString(forKey: key, value: missing, table: nil) != missing
  }
}
