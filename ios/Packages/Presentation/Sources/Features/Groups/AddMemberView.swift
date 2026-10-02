import Domain
import SwiftUI

/// The Add Member form (FR-M1, FR-M2).
struct AddMemberView: View {
  @Bindable var viewModel: AddMemberViewModel
  let onAdded: (Member) -> Void
  @Environment(\.dismiss) private var dismiss

  var body: some View {
    Form {
      Section {
        TextField(text: $viewModel.displayName) {
          Text(LocalizedStringKey(Self.displayName), bundle: .module)
        }
        .textContentType(.name)
        FieldErrorText(key: viewModel.errors["display_name"])
        TextField(text: $viewModel.email, prompt: Text(LocalizedStringKey(Self.emailPrompt), bundle: .module)) {
          Text(LocalizedStringKey(Self.email), bundle: .module)
        }
        .textContentType(.emailAddress)
        .emailEntry()
        FieldErrorText(key: viewModel.errors["email"])
      } footer: {
        Text(LocalizedStringKey(Self.footer), bundle: .module)
      }
      Section {
        FormMessage(key: viewModel.errors.message)
        SubmitButton(titleKey: Self.add, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit) {
          if let member = await viewModel.submit() { onAdded(member) }
        }
      }
    }
    .navigationTitle(Text(LocalizedStringKey(Self.title), bundle: .module))
    .toolbar {
      ToolbarItem(placement: .cancellationAction) {
        Button {
          dismiss()
        } label: {
          Text(LocalizedStringKey(CommonKeys.cancel), bundle: .module)
        }
      }
    }
  }

  nonisolated static let title = "Add Member"
  nonisolated static let displayName = "Name in this Group"
  nonisolated static let email = "Email (optional)"
  nonisolated static let emailPrompt = "bob@example.com"
  nonisolated static let footer =
    "Someone with a Splits account joins at once. Anyone else is added as a Placeholder you can record Expenses for; it becomes theirs when they sign up with this email."
  nonisolated static let add = "Add"
  nonisolated static let allKeys = [title, displayName, email, emailPrompt, footer, add]
}
