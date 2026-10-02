import Foundation

/// A person with an account (GLOSSARY: User).
public struct User: Equatable, Hashable, Sendable {
  public let id: UUID
  public let email: String
  public let emailVerified: Bool

  public init(id: UUID, email: String, emailVerified: Bool) {
    self.id = id
    self.email = email
    self.emailVerified = emailVerified
  }
}
