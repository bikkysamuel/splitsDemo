import Testing

@testable import Domain

/// NFR-R1: a retried submission reuses its key; new input gets a new one.
struct WriteKeyTests {
  @Test func theSameInputGetsTheSameKey() {
    var keys = WriteKeys<String>()

    #expect(keys.key(for: "Trip") == keys.key(for: "Trip"))
  }

  @Test func changedInputGetsANewKey() {
    var keys = WriteKeys<String>()
    let first = keys.key(for: "Trip")

    #expect(keys.key(for: "Flat") != first)
  }

  @Test func aSucceededActionStartsOver() {
    var keys = WriteKeys<String>()
    let first = keys.key(for: "Trip")
    keys.succeeded()

    #expect(keys.key(for: "Trip") != first)
  }
}
