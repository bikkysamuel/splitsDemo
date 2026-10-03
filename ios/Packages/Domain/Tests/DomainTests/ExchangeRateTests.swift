import Foundation
import Testing

@testable import Domain

/// What the User types as an Exchange Rate becomes the exact decimal string
/// the server takes (FR-E5): positive, at most 2 decimal places (Q87). The
/// conversion itself is the server's (ADR-0006).
struct ExchangeRateTests {
  @Test(arguments: [
    ("83.25", "en_US", "83.25"),
    ("83,25", "de_DE", "83.25"),
    (" 0.55 ", "en_US", "0.55"),
    ("1,234.5", "en_US", "1234.5"),
    ("2", "en_US", "2"),
    // Stored exactly as entered (FR-E5): the typed scale is kept.
    ("83.20", "en_US", "83.20"),
    ("83,20", "de_DE", "83.20"),
    ("1.0", "en_US", "1.0"),
  ])
  func aRateIsSentAsADecimalWithAPoint(text: String, locale: String, expected: String) {
    #expect(ExchangeRate.input(from: text, locale: Locale(identifier: locale)) == expected)
  }

  @Test(arguments: ["83.255", "83.200", "0", "0.00", "-83", "abc", ""])
  func aMalformedRateIsNotSent(text: String) {
    #expect(ExchangeRate.input(from: text, locale: Locale(identifier: "en_US")) == nil)
  }

  @Test func aStoredRateIsShownInTheUsersLocale() {
    #expect(ExchangeRate.display("1234.5", locale: Locale(identifier: "de_DE")) == "1.234,5")
    #expect(ExchangeRate.display("83.25", locale: Locale(identifier: "en_US")) == "83.25")
    // Shown as stored, its scale kept.
    #expect(ExchangeRate.display("83.20", locale: Locale(identifier: "de_DE")) == "83,20")
  }
}
