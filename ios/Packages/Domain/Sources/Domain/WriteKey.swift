import Foundation

/// The Idempotency-Key of one User action (NFR-R1): a form's submission,
/// kept across retries of that same submission and replaced when the input
/// changes. The server replays a repeated key instead of acting twice.
public struct WriteKey: Equatable, Hashable, Sendable {
  public let value: UUID

  public init(_ value: UUID = UUID()) {
    self.value = value
  }
}

/// Hands out one `WriteKey` per distinct input: the same input gets the
/// same key (a retry), changed input a fresh one.
public struct WriteKeys<Input: Hashable & Sendable>: Sendable {
  private var last: (input: Input, key: WriteKey)?

  public init() {}

  public mutating func key(for input: Input) -> WriteKey {
    if let last, last.input == input { return last.key }
    let key = WriteKey()
    last = (input, key)
    return key
  }

  /// Forgets the key once its action succeeded, so the next one is new.
  public mutating func succeeded() { last = nil }
}
