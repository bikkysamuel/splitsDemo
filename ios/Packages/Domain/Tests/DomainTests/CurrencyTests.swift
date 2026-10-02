import Testing

@testable import Domain

/// The app offers exactly the currencies the server accepts (Q91).
struct CurrencyTests {
  @Test func activeCodesAreSortedUniqueThreeLetterCodes() {
    let codes = Currency.activeCodes

    #expect(codes == codes.sorted())
    #expect(Set(codes).count == codes.count)
    #expect(codes.allSatisfy { $0.count == 3 && $0.allSatisfy { $0.isUppercase && $0.isASCII } })
    #expect(codes.count > 150)
  }

  @Test(arguments: ["INR", "USD", "EUR", "JPY", "KWD"])
  func commonCurrenciesAreActive(code: String) {
    #expect(Currency.isActive(code))
  }

  @Test(arguments: ["XAU", "XXX", "ZZZ", "inr", ""])
  func fundsMetalsAndJunkAreNot(code: String) {
    #expect(!Currency.isActive(code))
  }
}
