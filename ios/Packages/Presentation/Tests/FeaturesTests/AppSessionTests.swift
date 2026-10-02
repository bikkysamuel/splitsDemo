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

  @Test func usingAnotherAccountForgetsTheSession() async {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    session.didAuthenticate(.unverified)

    await session.useAnotherAccount()

    #expect(session.state == .signedOut)
    #expect(await repository.forgotSession)
  }
}
