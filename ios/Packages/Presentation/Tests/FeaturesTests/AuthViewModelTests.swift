import Domain
import Testing

@testable import Features

@MainActor
struct SignUpViewModelTests {
  @Test func signingUpSendsTheUserToVerification() async {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    let viewModel = SignUpViewModel(session: session)
    viewModel.email = "alice@example.com"
    viewModel.password = "correct horse battery"

    await viewModel.submit()

    #expect(
      await repository.calls == [.init(name: "signUp", email: "alice@example.com", secret: "correct horse battery")])
    #expect(session.state == .needsVerification(.unverified))
    #expect(viewModel.errors == FormErrors())
    #expect(!viewModel.isSubmitting)
  }

  @Test func cannotSubmitWithAnEmptyField() async {
    let repository = FakeAuthRepository()
    let viewModel = SignUpViewModel(session: AppSession(repository: repository))
    viewModel.email = "alice@example.com"

    #expect(!viewModel.canSubmit)
    await viewModel.submit()
    #expect(await repository.calls.isEmpty)
  }

  @Test func refusedFieldsShowUnderEachField() async {
    let repository = FakeAuthRepository()
    await repository.set(
      signUp: .failure(
        .invalidFields([
          FieldIssue(field: "email", reason: .invalid), FieldIssue(field: "password", reason: .tooCommon),
        ])))
    let session = AppSession(repository: repository)
    let viewModel = SignUpViewModel(session: session)
    viewModel.email = "alice"
    viewModel.password = "password1234"

    await viewModel.submit()

    #expect(viewModel.errors.email == "Enter a valid email address.")
    #expect(viewModel.errors.password == "This password is too common. Choose another.")
    #expect(viewModel.errors.message == ServiceErrorMessage.checkFields)
    #expect(session.state == .launching)
  }

  @Test func aTakenEmailSuggestsSigningIn() async {
    let repository = FakeAuthRepository()
    await repository.set(signUp: .failure(.problem(.emailTaken)))
    let viewModel = SignUpViewModel(session: AppSession(repository: repository))
    viewModel.email = "alice@example.com"
    viewModel.password = "correct horse battery"

    await viewModel.submit()

    #expect(viewModel.errors.message == "An account with this email already exists. Sign in instead.")
    #expect(viewModel.errors.email == nil)
  }

  @Test func aSuccessfulRetryClearsEarlierErrors() async {
    let repository = FakeAuthRepository()
    await repository.set(signUp: .failure(.unreachable))
    let viewModel = SignUpViewModel(session: AppSession(repository: repository))
    viewModel.email = "alice@example.com"
    viewModel.password = "correct horse battery"
    await viewModel.submit()
    #expect(viewModel.errors.message == ServiceErrorMessage.unreachable)

    await repository.set(signUp: .success(.unverified))
    await viewModel.submit()

    #expect(viewModel.errors == FormErrors())
  }
}

@MainActor
struct SignInViewModelTests {
  @Test func aVerifiedUserGoesHome() async {
    let repository = FakeAuthRepository()
    let session = AppSession(repository: repository)
    let viewModel = SignInViewModel(session: session)
    viewModel.email = "alice@example.com"
    viewModel.password = "correct horse battery"

    await viewModel.submit()

    #expect(session.state == .signedIn(.verified))
  }

  @Test func anUnverifiedUserGoesToVerification() async {
    let repository = FakeAuthRepository()
    await repository.set(signIn: .success(.unverified))
    let session = AppSession(repository: repository)
    let viewModel = SignInViewModel(session: session)
    viewModel.email = "alice@example.com"
    viewModel.password = "correct horse battery"

    await viewModel.submit()

    #expect(session.state == .needsVerification(.unverified))
  }

  // FR-U2: a 401 from sign-in is a form error.
  @Test func wrongCredentialsAreAFormError() async {
    let repository = FakeAuthRepository()
    await repository.set(signIn: .failure(.problem(.invalidCredentials)))
    let session = AppSession(repository: repository)
    let viewModel = SignInViewModel(session: session)
    viewModel.email = "alice@example.com"
    viewModel.password = "wrong password!!"

    await viewModel.submit()

    #expect(viewModel.errors.message == "The email or password is wrong.")
    #expect(session.state == .launching)
  }
}

@MainActor
struct VerifyEmailViewModelTests {
  private func makeViewModel(_ repository: FakeAuthRepository) -> (VerifyEmailViewModel, AppSession) {
    let session = AppSession(repository: repository)
    session.didAuthenticate(.unverified)
    return (VerifyEmailViewModel(email: "alice@example.com", session: session), session)
  }

  @Test func theRightCodeSignsTheVerifiedUserIn() async {
    let repository = FakeAuthRepository()
    let (viewModel, session) = makeViewModel(repository)
    viewModel.code = "123456"

    await viewModel.submit()

    #expect(await repository.calls == [.init(name: "verifyEmail", email: "alice@example.com", secret: "123456")])
    #expect(session.state == .signedIn(.verified))
  }

  @Test func theCodeKeepsSixDigitsOnly() {
    let (viewModel, _) = makeViewModel(FakeAuthRepository())

    viewModel.code = "12 3-45678"

    #expect(viewModel.code == "123456")
  }

  @Test func cannotSubmitFewerThanSixDigits() async {
    let repository = FakeAuthRepository()
    let (viewModel, _) = makeViewModel(repository)
    viewModel.code = "12345"

    #expect(!viewModel.canSubmit)
    await viewModel.submit()
    #expect(await repository.calls.isEmpty)
  }

  @Test func aWrongCodeIsShownAndKeepsTheUserHere() async {
    let repository = FakeAuthRepository()
    await repository.set(verify: .failure(.problem(.invalidCode)))
    let (viewModel, session) = makeViewModel(repository)
    viewModel.code = "000000"

    await viewModel.submit()

    #expect(viewModel.message == "That code didn't work. Check it, or send a new one.")
    #expect(viewModel.messageIsError)
    #expect(session.state == .needsVerification(.unverified))
  }

  @Test func resendingConfirmsAndClearsTheCode() async {
    let repository = FakeAuthRepository()
    let (viewModel, _) = makeViewModel(repository)
    viewModel.code = "000000"

    await viewModel.resend()

    #expect(await repository.calls == [.init(name: "resend", email: "alice@example.com", secret: "")])
    #expect(viewModel.message == VerifyEmailViewModel.resentMessage)
    #expect(!viewModel.messageIsError)
    #expect(viewModel.code.isEmpty)
  }

  @Test func aFailedResendIsShown() async {
    let repository = FakeAuthRepository()
    await repository.set(resend: .failure(.unreachable))
    let (viewModel, _) = makeViewModel(repository)

    await viewModel.resend()

    #expect(viewModel.message == ServiceErrorMessage.unreachable)
    #expect(viewModel.messageIsError)
  }

  @Test func usingAnotherAccountReturnsToSignIn() async {
    let repository = FakeAuthRepository()
    let (viewModel, session) = makeViewModel(repository)

    await viewModel.useAnotherAccount()

    #expect(session.state == .signedOut)
  }
}
