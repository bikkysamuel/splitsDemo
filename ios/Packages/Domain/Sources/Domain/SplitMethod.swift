import Foundation

/// How an Expense is divided among the Members of its Split (GLOSSARY:
/// Split, FR-E2). The server computes the Shares (ADR-0006); the app only
/// reads what the User typed into the exact decimal string the server takes.
public enum SplitMethod: String, CaseIterable, Hashable, Sendable {
  /// Equally among the Members.
  case equal
  /// Exact amounts that must sum to the total.
  case exact
  /// Percentages with at most 2 decimal places that must sum to exactly 100.
  case percentage
  /// Whole-number parts, such as 2:1:1.
  case ratio

  /// Whether each Member needs an entry.
  public var takesInput: Bool { self != .equal }

  /// The server's form of what the User typed for one Member, or nil when it
  /// isn't well-formed: for `exact`, minor units of `currency` ("25050");
  /// for `percentage`, a positive number with at most 2 decimal places, with
  /// a point ("33.33"); for `ratio`, a positive whole number ("2"). Whether
  /// the entries add up is the server's check.
  public func input(from text: String, currency: String, locale: Locale = .current) -> String? {
    switch self {
    case .equal:
      return nil
    case .exact:
      return Money.parse(text, currency: currency, locale: locale).map { String($0.minorUnits) }
    case .percentage:
      return DecimalEntry.positive(text, places: 2, locale: locale)
    case .ratio:
      return DecimalEntry.positive(text, places: 0, locale: locale)
    }
  }

  /// A stored entry ("33.33", "2") as the User's locale writes it, such as
  /// "33,33 %"; an exact amount is shown by its Share instead.
  public func displayInput(_ input: String, locale: Locale = .current) -> String {
    guard let value = Decimal(string: input, locale: Locale(identifier: "en_US_POSIX")) else { return input }
    switch self {
    case .percentage: return value.formatted(.percent.scale(1).locale(locale))
    default: return value.formatted(.number.locale(locale))
    }
  }
}

/// One Member of a Split and what was entered for them, in the server's
/// form (see `SplitMethod.input(from:currency:locale:)`); nil for an equal
/// Split.
public struct SplitEntry: Equatable, Hashable, Sendable {
  public let memberID: UUID
  public let input: String?

  public init(memberID: UUID, input: String? = nil) {
    self.memberID = memberID
    self.input = input
  }
}
