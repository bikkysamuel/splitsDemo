/// The currencies a Group or Expense may use: the active ISO 4217 codes the
/// server accepts (Q91). Keep in step with `activeCurrencies` in
/// server/internal/ledger/currency.go; `CurrencyTests` checks the shape and
/// the server refuses anything else.
public enum Currency {
  public static let activeCodes: [String] = [
    "AED", "AFN", "ALL", "AMD", "AOA", "ARS", "AUD", "AWG", "AZN", "BAM", "BBD", "BDT", "BHD", "BIF", "BMD",
    "BND", "BOB", "BRL", "BSD", "BTN", "BWP", "BYN", "BZD", "CAD", "CDF", "CHF", "CLP", "CNY", "COP", "CRC",
    "CUP", "CVE", "CZK", "DJF", "DKK", "DOP", "DZD", "EGP", "ERN", "ETB", "EUR", "FJD", "FKP", "GBP", "GEL",
    "GHS", "GIP", "GMD", "GNF", "GTQ", "GYD", "HKD", "HNL", "HTG", "HUF", "IDR", "ILS", "INR", "IQD", "IRR",
    "ISK", "JMD", "JOD", "JPY", "KES", "KGS", "KHR", "KMF", "KPW", "KRW", "KWD", "KYD", "KZT", "LAK", "LBP",
    "LKR", "LRD", "LSL", "LYD", "MAD", "MDL", "MGA", "MKD", "MMK", "MNT", "MOP", "MRU", "MUR", "MVR", "MWK",
    "MXN", "MYR", "MZN", "NAD", "NGN", "NIO", "NOK", "NPR", "NZD", "OMR", "PAB", "PEN", "PGK", "PHP", "PKR",
    "PLN", "PYG", "QAR", "RON", "RSD", "RUB", "RWF", "SAR", "SBD", "SCR", "SDG", "SEK", "SGD", "SHP", "SLE",
    "SOS", "SRD", "SSP", "STN", "SVC", "SYP", "SZL", "THB", "TJS", "TMT", "TND", "TOP", "TRY", "TTD", "TWD",
    "TZS", "UAH", "UGX", "USD", "UYU", "UZS", "VED", "VES", "VND", "VUV", "WST", "XAF", "XCD", "XCG", "XOF",
    "XPF", "YER", "ZAR", "ZMW", "ZWG",
  ]

  private static let activeSet = Set(activeCodes)

  public static func isActive(_ code: String) -> Bool { activeSet.contains(code) }

  /// Decimal places of the minor unit (ISO 4217): 2 for INR, 0 for JPY, 3
  /// for KWD. Same table as `MinorUnits` in server/internal/ledger.
  public static func minorUnitDigits(_ code: String) -> Int { minorUnitExceptions[code] ?? 2 }

  private static let minorUnitExceptions: [String: Int] = [
    "BIF": 0, "CLP": 0, "DJF": 0, "GNF": 0, "ISK": 0, "JPY": 0, "KMF": 0, "KRW": 0, "PYG": 0, "RWF": 0, "UGX": 0,
    "UYI": 0, "VND": 0, "VUV": 0, "XAF": 0, "XOF": 0, "XPF": 0, "BHD": 3, "IQD": 3, "JOD": 3, "KWD": 3, "LYD": 3,
    "OMR": 3, "TND": 3, "CLF": 4, "UYW": 4,
  ]
}
