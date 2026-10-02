import SwiftUI

/// Email and password sign-up (FR-A1, FR-A3).
struct SignUpView: View {
  @Bindable var viewModel: SignUpViewModel

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
        .textContentType(.newPassword)
        FieldErrorText(key: viewModel.errors.password)
      } footer: {
        Text(LocalizedStringKey(Keys.newPasswordHint), bundle: .module)
      }
      Section {
        FormMessage(key: viewModel.errors.message)
        SubmitButton(
          titleKey: Keys.signUp, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit
        ) {
          await viewModel.submit()
        }
      }
    }
    .navigationTitle(Text(LocalizedStringKey(Keys.signUp), bundle: .module))
  }
}
