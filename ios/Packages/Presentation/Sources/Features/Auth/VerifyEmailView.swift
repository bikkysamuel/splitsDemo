import SwiftUI

/// The 6-digit code screen (FR-A2), with resend and a way out to another
/// account.
struct VerifyEmailView: View {
  @Bindable var viewModel: VerifyEmailViewModel

  var body: some View {
    Form {
      Section {
        TextField(text: $viewModel.code, prompt: Text(verbatim: "123456")) {
          Text(LocalizedStringKey(Keys.code), bundle: .module)
        }
        .textContentType(.oneTimeCode)
        .font(.title2.monospacedDigit())
        .codeEntry()
      } header: {
        Text(LocalizedStringKey("We sent a 6-digit code to \(viewModel.email)."), bundle: .module)
          .textCase(nil)
      }
      Section {
        FormMessage(key: viewModel.message, isError: viewModel.messageIsError)
        SubmitButton(
          titleKey: Keys.verify, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit
        ) {
          await viewModel.submit()
        }
      }
      Section {
        Button {
          Task { await viewModel.resend() }
        } label: {
          Text(LocalizedStringKey(Keys.resend), bundle: .module)
            .frame(minHeight: 44)
        }
        .disabled(viewModel.isResending)
        Button {
          Task { await viewModel.useAnotherAccount() }
        } label: {
          Text(LocalizedStringKey(Keys.useAnotherAccount), bundle: .module)
            .frame(minHeight: 44)
        }
      }
    }
    .navigationTitle(Text(LocalizedStringKey(Keys.verifyTitle), bundle: .module))
  }
}

extension View {
  fileprivate func codeEntry() -> some View {
    #if os(iOS)
      self.keyboardType(.numberPad)
    #else
      self
    #endif
  }
}
