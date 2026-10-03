import Foundation

/// The rate a Member enters to convert an Expense's Original Amount into the
/// Group Currency: units of the Group Currency per one unit of the Expense's
/// currency (GLOSSARY: Exchange Rate, FR-E5). The app only reads what the
/// User typed into the exact decimal string the server takes; the conversion
/// is the server's (ADR-0006, ADR-0007).
public enum ExchangeRate {
  /// The most decimal places a rate may have (Q87).
  public static let maxDecimals = 2

  /// The server's form of a typed rate ("83.25", with a point), or nil when
  /// it isn't a positive number with at most 2 decimal places. The typed
  /// scale is kept ("83,20" → "83.20"), so the rate is stored as entered.
  public static func input(from text: String, locale: Locale = .current) -> String? {
    guard var value = DecimalEntry.positive(text, places: maxDecimals, locale: locale) else { return nil }
    let typed = typedDecimals(text, locale: locale)
    // The server takes at most 2 places as written, even "83.200".
    guard typed <= maxDecimals else { return nil }
    let kept = value.split(separator: ".").dropFirst().first?.count ?? 0
    if typed > kept {
      value += (kept == 0 ? "." : "") + String(repeating: "0", count: typed - kept)
    }
    return value
  }

  /// How many digits were typed after the locale's decimal separator.
  private static func typedDecimals(_ text: String, locale: Locale) -> Int {
    let separator = locale.decimalSeparator ?? "."
    guard let range = text.range(of: separator, options: .backwards) else { return 0 }
    return text[range.upperBound...].prefix { $0.isNumber }.count
  }

  /// A stored rate as editable text in the User's locale, its scale kept
  /// and without grouping ("1234,5", "83.20"), to pre-fill a form;
  /// `input(from:locale:)` reads it back.
  public static func editableText(_ rate: String, locale: Locale = .current) -> String {
    guard let value = Decimal(string: rate, locale: Locale(identifier: "en_US_POSIX")) else { return rate }
    let scale = rate.split(separator: ".").dropFirst().first?.count ?? 0
    return value.formatted(.number.grouping(.never).precision(.fractionLength(scale)).locale(locale))
  }

  /// A stored rate ("1234.5", "83.20") as the User's locale writes it,
  /// its scale kept ("1.234,5", "83,20").
  public static func display(_ rate: String, locale: Locale = .current) -> String {
    guard let value = Decimal(string: rate, locale: Locale(identifier: "en_US_POSIX")) else { return rate }
    let scale = rate.split(separator: ".").dropFirst().first?.count ?? 0
    return value.formatted(.number.precision(.fractionLength(scale)).locale(locale))
  }
}

/// Reads typed decimals into the server's exact decimal strings.
enum DecimalEntry {
  /// `text` as a positive decimal with at most `places` decimal places,
  /// written with a point and no grouping, or nil.
  static func positive(_ text: String, places: Int, locale: Locale) -> String? {
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty, let value = try? Decimal(trimmed, format: .number.locale(locale)), value > 0 else {
      return nil
    }
    var original = value
    var rounded = Decimal()
    NSDecimalRound(&rounded, &original, places, .plain)
    guard rounded == value else { return nil }
    // Decimal's description always uses a point and no grouping.
    return value.description
  }
}
