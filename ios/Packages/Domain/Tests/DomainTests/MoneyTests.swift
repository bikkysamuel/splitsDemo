import Foundation
import Testing

@testable import Domain

/// Money formats and parses only (ADR-0006); the server does the maths.
struct MoneyTests {
  @Test(arguments: [
    (Money(minorUnits: 100001, currency: "INR"), "en_IN", "₹1,000.01"),
    (Money(minorUnits: 1500, currency: "JPY"), "en_US", "¥1,500"),
    (Money(minorUnits: 5, currency: "USD"), "en_US", "$0.05"),
  ])
  func formatsWithTheCurrencysDecimalPlaces(money: Money, locale: String, expected: String) {
    #expect(money.formatted(locale: Locale(identifier: locale)) == expected)
  }

  // Three decimal places for the fils (the label's spacing varies by OS).
  @Test func formatsThreeDecimalPlacesForKWD() {
    #expect(Money(minorUnits: 1234, currency: "KWD").formatted(locale: Locale(identifier: "en_US")).hasSuffix("1.234"))
  }

  @Test(arguments: [
    ("1000.01", "INR", "en_IN", Int64(100001)),
    ("1,000.5", "INR", "en_US", 100050),
    ("1000,50", "EUR", "de_DE", 100050),
    ("1500", "JPY", "en_US", 1500),
    ("1.234", "KWD", "en_US", 1234),
    (" 7 ", "USD", "en_US", 700),
  ])
  func parsesTypedAmountsIntoMinorUnits(text: String, currency: String, locale: String, minor: Int64) {
    #expect(Money.parse(text, currency: currency, locale: Locale(identifier: locale))?.minorUnits == minor)
  }

  @Test(arguments: [
    ("", "INR"), ("abc", "INR"), ("0", "INR"), ("-5", "INR"), ("1.234", "INR"), ("1.5", "JPY"),
  ])
  func refusesWhatIsntAPositiveAmountInTheCurrency(text: String, currency: String) {
    #expect(Money.parse(text, currency: currency, locale: Locale(identifier: "en_US")) == nil)
  }
}
