import Domain
import Foundation
import Observation

/// Add Member (FR-M1, FR-M2): a display name and, optionally, an email. A
/// verified User with that email joins at once; anyone else becomes a
/// Placeholder Member.
@MainActor
@Observable
public final class AddMemberViewModel {
  public let groupID: UUID
  public var displayName = ""
  public var email = ""
  public private(set) var isSubmitting = false
  private(set) var errors = FormErrors()

  private let repository: any GroupsRepository
  private var keys = WriteKeys<[String]>()

  public init(groupID: UUID, repository: any GroupsRepository) {
    self.groupID = groupID
    self.repository = repository
  }

  public var canSubmit: Bool {
    !displayName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isSubmitting
  }

  /// Adds the Member; returns it on success.
  public func submit() async -> Member? {
    guard canSubmit else { return nil }
    isSubmitting = true
    defer { isSubmitting = false }
    let trimmed = email.trimmingCharacters(in: .whitespacesAndNewlines)
    let key = keys.key(for: [displayName, trimmed])
    do {
      let member = try await repository.addMember(
        groupID: groupID, displayName: displayName, email: trimmed.isEmpty ? nil : trimmed, key: key)
      keys.succeeded()
      errors = FormErrors()
      return member
    } catch {
      errors = FormErrors(error)
      return nil
    }
  }
}

extension AddMemberViewModel: Identifiable {
  public nonisolated var id: ObjectIdentifier { ObjectIdentifier(self) }
}
