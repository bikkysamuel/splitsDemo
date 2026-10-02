import Foundation

/// An amount in integer minor units of an ISO 4217 currency (ADR-0002).
/// It formats and parses only: every calculation happens on the server
/// (ADR-0006), so `Money` has no arithmetic.
public struct Money: Equatable, Hashable, Sendable {
  public let minorUnits: Int64
  public let currency: String

  public init(minorUnits: Int64, currency: String) {
    self.minorUnits = minorUnits
    self.currency = currency
  }

  /// The amount as the User's locale writes it, such as "₹1,000.01".
  public func formatted(locale: Locale = .current) -> String {
    decimalValue.formatted(
      .currency(code: currency).locale(locale).precision(.fractionLength(Currency.minorUnitDigits(currency))))
  }

  /// The amount in major units, for display only.
  var decimalValue: Decimal {
    Decimal(minorUnits) / pow(10, Currency.minorUnitDigits(currency))
  }

  /// Reads an amount the User typed ("1,000.5", "1000,50" in de_DE) into
  /// minor units. Returns nil when it isn't a positive number with at most
  /// the currency's decimal places.
  public static func parse(_ text: String, currency: String, locale: Locale = .current) -> Money? {
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty,
      let value = try? Decimal(trimmed, format: .number.locale(locale)), value > 0
    else { return nil }
    let digits = Currency.minorUnitDigits(currency)
    var scaled = value * pow(10, digits)
    var rounded = Decimal()
    NSDecimalRound(&rounded, &scaled, 0, .plain)
    guard rounded == scaled, rounded <= Decimal(Int64.max) else { return nil }
    return Money(minorUnits: NSDecimalNumber(decimal: rounded).int64Value, currency: currency)
  }
}
