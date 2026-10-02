import Domain
import Testing

@testable import Features

/// FR-U1: the launch check decides the first screen.
@MainActor
struct AppSessionTests {
  @Test func startsLaunching() {
    let session = AppSession(repository: FakeAuthRepository())

    #expect(session.state == .launching)
  }

  @Test func withoutASavedSessionGoesToSignIn() async {
    let session = AppSession(repository: FakeAuthRepository())

    await session.start()

    #expect(session.state == .signedOut)
  }

  @Test func aVerifiedUserGoesHome() async {
    let repository = FakeAuthRepository()
    await repository.set(currentUser: .success(.verified))
    let session = AppSession(repository: repository)

    await session.start()

    #expect(session.state == .signedIn(.verified))
  }

  @Test func anUnverifiedUserGoesToVerification() async {
    let repository = FakeAuthRepository()
    await repository.set(currentUser: .success(.unverified))
    let session = AppSession(repository: repository)

    await session.start()

    #expect(session.state == .needsVerification(.unverified))
  }

  // FR-U1: no connection must never send a signed-in User to sign-in.
  @Test func aFailedCheckStaysOnTheSplashWithTheError() async {
    let repository = FakeAuthRepository()
    await repository.set(currentUser: .failure(.unreachable))
    let session = AppSession(repository: repository)

    await session.start()

    #expect(session.state == .launching)
    #expect(session.launchError == .unreachable)
  }

  @Test func retryingAfterAFailureClearsTheErrorAndRoutes() async {
    let repository = FakeAuthRepository()
    await repository.set(currentUser: .failure(.unreachable))
    let session = AppSession(repository: repository)
    await session.start()

    await repository.set(currentUser: .success(.verified))
    await session.start()

    #expect(session.launchError == nil)
    #expect(session.state == .signedIn(.verified))
  }

  @Test(arguments: [
    (User.verified, AppSession.State.signedIn(.verified)), (.unverified, .needsVerification(.unverified)),
  ])
  func authenticatingRoutesByVerification(user: User, expected: AppSession.State) {
    let session = AppSession(repository: FakeAuthRepository())

    session.didAuthenticate(user)

    #expect(session.state == expected)
  }

  @Test func usingAnotherAccountSignsOut() async {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    session.didAuthenticate(.unverified)

    await session.useAnotherAccount()

    #expect(session.state == .signedOut)
    #expect(await repository.signedOut)
  }

  @Test func signingOutReturnsToSignIn() async {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    session.didAuthenticate(.verified)

    await session.signOut()

    #expect(session.state == .signedOut)
    #expect(await repository.signedOut)
    #expect(!session.showsSessionExpired)
  }

  // FR-U2: a failed refresh shows one "session expired" alert and returns
  // to sign-in.
  @Test func anExpiredSessionShowsTheAlertAndReturnsToSignIn() async throws {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    session.didAuthenticate(.verified)
    let observing = Task { await session.observeSessionExpirations() }
    defer { observing.cancel() }

    repository.expireSession()
    try await waitUntil { session.state == .signedOut }

    #expect(session.showsSessionExpired)
    session.dismissSessionExpired()
    #expect(!session.showsSessionExpired)
  }

  // Already signed out (say, the User signed out while a request failed):
  // no alert.
  @Test func anExpiryAfterSigningOutShowsNoAlert() async throws {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    await session.start()
    let observing = Task { await session.observeSessionExpirations() }
    defer { observing.cancel() }

    repository.expireSession()
    try await Task.sleep(for: .milliseconds(50))

    #expect(!session.showsSessionExpired)
  }

  private func waitUntil(_ condition: () -> Bool) async throws {
    for _ in 0..<100 where !condition() {
      try await Task.sleep(for: .milliseconds(10))
    }
    #expect(condition())
  }
}
