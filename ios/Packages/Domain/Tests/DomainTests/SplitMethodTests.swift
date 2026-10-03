import Foundation
import Testing

@testable import Domain

/// What the User types for one Member becomes the exact decimal string the
/// server takes (FR-E2). The app checks well-formedness only; sums are the
/// server's job (ADR-0006).
struct SplitMethodTests {
  private let us = Locale(identifier: "en_US")
  private let de = Locale(identifier: "de_DE")

  @Test(arguments: [
    ("250.50", "INR", "en_US", "25050"),
    ("1.000,5", "EUR", "de_DE", "100050"),
    ("1500", "JPY", "en_US", "1500"),
  ])
  func anExactAmountIsSentInMinorUnits(text: String, currency: String, locale: String, expected: String) {
    #expect(SplitMethod.exact.input(from: text, currency: currency, locale: Locale(identifier: locale)) == expected)
  }

  @Test(arguments: [
    ("33.33", "en_US", "33.33"),
    ("33,3", "de_DE", "33.3"),
    ("50", "en_US", "50"),
    ("25%", "en_US", "25"),
    ("12,5 %", "de_DE", "12.5"),
  ])
  func aPercentageIsSentWithAPoint(text: String, locale: String, expected: String) {
    #expect(SplitMethod.percentage.input(from: text, currency: "INR", locale: Locale(identifier: locale)) == expected)
  }

  @Test func aRatioPartIsAWholeNumber() {
    #expect(SplitMethod.ratio.input(from: " 2 ", currency: "INR", locale: us) == "2")
  }

  @Test(arguments: [
    (SplitMethod.exact, "12.345"), (.exact, "0"), (.exact, "-5"), (.exact, "abc"),
    (.percentage, "33.333"), (.percentage, "0"), (.percentage, "-10"),
    (.ratio, "1.5"), (.ratio, "0"), (.ratio, ""),
  ])
  func aMalformedInputIsRefused(method: SplitMethod, text: String) {
    #expect(method.input(from: text, currency: "INR", locale: us) == nil)
  }

  @Test func anEqualSplitTakesNoInput() {
    #expect(SplitMethod.equal.input(from: "2", currency: "INR", locale: us) == nil)
    #expect(!SplitMethod.equal.takesInput)
    #expect(SplitMethod.ratio.takesInput)
  }

  @Test func aPercentageIsDisplayedInTheUsersLocale() {
    #expect(SplitMethod.percentage.displayInput("33.5", locale: de).contains("33,5"))
    #expect(SplitMethod.ratio.displayInput("2", locale: us) == "2")
  }

  /// Editing an Expense pre-fills each entry as typed text that reads back
  /// to the same stored entry (FR-E6).
  @Test(arguments: [
    (SplitMethod.exact, "25050", "250,50"), (.percentage, "33.5", "33,5"), (.ratio, "2", "2"),
    (.exact, "1234567", "12345,67"),
  ])
  func aStoredEntryBecomesEditableTextThatReadsBack(method: SplitMethod, stored: String, expected: String) {
    #expect(method.editableInput(stored, currency: "EUR", locale: de) == expected)
    #expect(method.input(from: expected, currency: "EUR", locale: de) == stored)
  }
}
