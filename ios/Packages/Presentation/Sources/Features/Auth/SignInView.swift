import SwiftUI

/// Email and password sign-in (FR-A4), with a link to sign-up.
struct SignInView<SignUp: View>: View {
  @Bindable var viewModel: SignInViewModel
  @ViewBuilder let signUp: () -> SignUp

  var body: some View {
    Form {
      Section {
        TextField(text: $viewModel.email, prompt: Text(LocalizedStringKey(Keys.emailPrompt), bundle: .module)) {
          Text(LocalizedStringKey(Keys.email), bundle: .module)
        }
        .textContentType(.username)
        .emailEntry()
        FieldErrorText(key: viewModel.errors.email)

        SecureField(text: $viewModel.password) {
          Text(LocalizedStringKey(Keys.password), bundle: .module)
        }
        .textContentType(.password)
        FieldErrorText(key: viewModel.errors.password)
      }
      Section {
        FormMessage(key: viewModel.errors.message)
        SubmitButton(
          titleKey: Keys.signIn, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit
        ) {
          await viewModel.submit()
        }
      }
      Section {
        NavigationLink {
          signUp()
        } label: {
          Text(LocalizedStringKey(Keys.createAccount), bundle: .module)
        }
      }
    }
    .navigationTitle(Text(LocalizedStringKey(Keys.signIn), bundle: .module))
  }
}

/// String Catalog keys shared by the auth screens.
enum Keys {
  static let email = "Email"
  static let emailPrompt = "you@example.com"
  static let password = "Password"
  static let newPasswordHint = "At least 10 characters."
  static let signIn = "Sign in"
  static let signUp = "Create account"
  static let createAccount = "New to Splits? Create an account"
  static let verifyTitle = "Verify your email"
  static let code = "Verification code"
  static let verify = "Verify"
  static let resend = "Send a new code"
  static let useAnotherAccount = "Use another account"
  static let codeSentFormat = "We sent a 6-digit code to %@."

  static let all = [
    email, emailPrompt, password, newPasswordHint, signIn, signUp, createAccount, verifyTitle, code, verify, resend,
    useAnotherAccount, codeSentFormat,
  ]
}

extension View {
  /// Email keyboard, no autocapitalization or autocorrection.
  func emailEntry() -> some View {
    #if os(iOS)
      self.keyboardType(.emailAddress)
        .textInputAutocapitalization(.never)
        .autocorrectionDisabled()
    #else
      self.autocorrectionDisabled()
    #endif
  }
}
