import Domain
import Observation

/// Create Group (FR-G1): a name, a Group Currency pre-filled from Settings,
/// and the User's own name in the Group.
@MainActor
@Observable
public final class CreateGroupViewModel {
  public var name = ""
  public var currency: String
  public var displayName = ""
  public private(set) var isSubmitting = false
  private(set) var errors = FormErrors()

  private let repository: any GroupsRepository
  private var keys = WriteKeys<[String]>()

  public init(repository: any GroupsRepository, preferences: any PreferencesRepository) {
    self.repository = repository
    self.currency = preferences.defaultCurrency()
  }

  public var canSubmit: Bool {
    !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
      && !displayName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isSubmitting
  }

  /// Creates the Group; returns it on success. Submitting the same input
  /// again after a failure reuses its Idempotency-Key, so a create whose
  /// answer was lost is replayed, not repeated.
  public func submit() async -> Domain.Group? {
    guard canSubmit else { return nil }
    isSubmitting = true
    defer { isSubmitting = false }
    let key = keys.key(for: [name, currency, displayName])
    do {
      let group = try await repository.createGroup(
        name: name, currency: currency, displayName: displayName, key: key)
      keys.succeeded()
      errors = FormErrors()
      return group
    } catch {
      errors = FormErrors(error)
      return nil
    }
  }
}
