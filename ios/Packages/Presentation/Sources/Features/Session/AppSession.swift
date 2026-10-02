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
  /// The one "session expired" alert after a failed refresh (FR-U2).
  public private(set) var showsSessionExpired = false

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

  /// Leaves the verification screen for sign-in with another account.
  func useAnotherAccount() async {
    await signOut()
  }

  /// Signs out from Settings: the Session is revoked and forgotten.
  func signOut() async {
    await repository.signOut()
    state = .signedOut
  }

  /// Returns to sign-in with one alert each time a refresh fails. Runs for
  /// the app's lifetime (RootView's task).
  public func observeSessionExpirations() async {
    for await _ in repository.sessionExpirations {
      switch state {
      case .signedIn, .needsVerification:
        state = .signedOut
        showsSessionExpired = true
      case .launching, .signedOut:
        // A launch check routes by itself; nothing to interrupt.
        break
      }
    }
  }

  func dismissSessionExpired() {
    showsSessionExpired = false
  }
}
