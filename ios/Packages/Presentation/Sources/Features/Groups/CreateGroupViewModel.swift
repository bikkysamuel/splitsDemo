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

  public init(repository: any GroupsRepository, preferences: any PreferencesRepository) {
    self.repository = repository
    self.currency = preferences.defaultCurrency()
  }

  public var canSubmit: Bool {
    !name.trimmingCharacters(in: .whitespaces).isEmpty
      && !displayName.trimmingCharacters(in: .whitespaces).isEmpty && !isSubmitting
  }

  /// Creates the Group; returns it on success.
  public func submit() async -> Domain.Group? {
    guard canSubmit else { return nil }
    isSubmitting = true
    defer { isSubmitting = false }
    do {
      let group = try await repository.createGroup(name: name, currency: currency, displayName: displayName)
      errors = FormErrors()
      return group
    } catch {
      errors = FormErrors(error)
      return nil
    }
  }
}
