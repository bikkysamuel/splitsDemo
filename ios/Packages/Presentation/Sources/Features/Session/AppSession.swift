import Domain
import Observation

/// The app-level session state that drives the root view (doc 05,
/// FR-U1): the splash while the saved Session is checked, then sign-in,
/// verification or Home.
@MainActor
@Observable
public final class AppSession {
  public enum State: Equatable, Sendable {
    case launching
    case signedOut
    case needsVerification(User)
    case signedIn(User)
  }

  public private(set) var state: State = .launching
  /// Why the launch check failed; the splash offers a retry. The state stays
  /// `launching`, so a valid Session is never thrown away (FR-U1).
  public private(set) var launchError: ServiceError?

  let repository: any AuthRepository

  public init(repository: any AuthRepository) {
    self.repository = repository
  }

  /// Checks the saved Session with the server and routes.
  public func start() async {
    state = .launching
    launchError = nil
    do {
      if let user = try await repository.currentUser() {
        didAuthenticate(user)
      } else {
        state = .signedOut
      }
    } catch {
      launchError = error
    }
  }

  /// Routes a User who just signed up, verified or signed in.
  func didAuthenticate(_ user: User) {
    state = user.emailVerified ? .signedIn(user) : .needsVerification(user)
  }

  /// Leaves the verification screen for sign-in, forgetting the unverified
  /// Session on this device.
  func useAnotherAccount() async {
    await repository.forgetSession()
    state = .signedOut
  }
}
