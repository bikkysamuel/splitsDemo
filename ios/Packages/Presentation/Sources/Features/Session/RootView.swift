import SwiftUI

/// The app's root: the session state picks the screen (doc 05, FR-U1).
public struct RootView: View {
  let session: AppSession

  public init(session: AppSession) {
    self.session = session
  }

  public var body: some View {
    Group {
      switch session.state {
      case .launching:
        SplashView(session: session)
      case .signedOut:
        SignedOutView(session: session)
      case .needsVerification(let user):
        NavigationStack {
          VerifyEmailView(viewModel: VerifyEmailViewModel(email: user.email, session: session))
        }
        // A new User gets a fresh screen, not the previous one's code.
        .id(user.id)
      case .signedIn:
        MainTabView()
      }
    }
    .task { await session.start() }
  }
}

/// Shown while the saved Session is checked. A failed check stays here
/// with a retry, never sign-in, so a valid Session is kept (FR-U1).
struct SplashView: View {
  let session: AppSession

  var body: some View {
    VStack(spacing: 16) {
      Text(LocalizedStringKey(Self.titleKey), bundle: .module)
        .font(.largeTitle.bold())
        .accessibilityAddTraits(.isHeader)
      if let error = session.launchError {
        Text(LocalizedStringKey(ServiceErrorMessage.key(for: error)), bundle: .module)
          .multilineTextAlignment(.center)
          .foregroundStyle(.secondary)
        Button {
          Task { await session.start() }
        } label: {
          Text(LocalizedStringKey(Self.retryKey), bundle: .module)
            .frame(minHeight: 44)  // NFR-A2: at least 44×44 pt
        }
        .buttonStyle(.borderedProminent)
      } else {
        ProgressView()
          .accessibilityLabel(Text(LocalizedStringKey(Self.checkingKey), bundle: .module))
      }
    }
    .padding()
  }

  nonisolated static let titleKey = "Splits"
  nonisolated static let retryKey = "Try again"
  nonisolated static let checkingKey = "Checking your session…"
}

/// Sign-in, with sign-up one tap away.
struct SignedOutView: View {
  let session: AppSession
  @State private var signIn: SignInViewModel
  @State private var signUp: SignUpViewModel

  init(session: AppSession) {
    self.session = session
    _signIn = State(initialValue: SignInViewModel(session: session))
    _signUp = State(initialValue: SignUpViewModel(session: session))
  }

  var body: some View {
    NavigationStack {
      SignInView(viewModel: signIn) {
        SignUpView(viewModel: signUp)
      }
    }
  }
}
