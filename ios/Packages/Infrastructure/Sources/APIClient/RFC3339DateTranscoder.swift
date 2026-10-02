import Foundation
import OpenAPIRuntime

/// RFC 3339 timestamps as the Go server writes them: fractional seconds
/// when non-zero, none otherwise. The runtime's default transcoder accepts
/// only the second form.
public struct RFC3339DateTranscoder: DateTranscoder {
  public init() {}

  public func encode(_ date: Date) throws -> String {
    date.formatted(Self.withFraction)
  }

  public func decode(_ text: String) throws -> Date {
    if let date = try? Self.withFraction.parse(text) { return date }
    if let date = try? Self.plain.parse(text) { return date }
    if let date = try? Self.withFraction.parse(Self.trimmedFraction(text)) { return date }
    throw DecodingError.dataCorrupted(.init(codingPath: [], debugDescription: "Not an RFC 3339 date: \(text)"))
  }

  private static let plain = Date.ISO8601FormatStyle()
  private static let withFraction = Date.ISO8601FormatStyle(includingFractionalSeconds: true)

  /// Cuts a fraction longer than milliseconds (Go writes nanoseconds) to
  /// three digits, which the format style parses.
  private static func trimmedFraction(_ text: String) -> String {
    guard let dot = text.firstIndex(of: ".") else { return text }
    let digits = text[text.index(after: dot)...].prefix { $0.isNumber }
    guard digits.count > 3 else { return text }
    let end = text.index(dot, offsetBy: digits.count + 1)
    return String(text[...dot]) + digits.prefix(3) + text[end...]
  }
}
