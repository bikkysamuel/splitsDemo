/// Checks whether the Splits server answers.
public protocol ServerHealthRepository: Sendable {
  /// Returns when the server is up; throws when it can't be reached.
  func checkHealth() async throws
}
