import Domain
import SwiftUI

/// The Create Group form (FR-G1).
struct CreateGroupView: View {
  @Bindable var viewModel: CreateGroupViewModel
  let onCreated: (Domain.Group) -> Void
  @Environment(\.dismiss) private var dismiss

  var body: some View {
    Form {
      Section {
        TextField(text: $viewModel.name, prompt: Text(LocalizedStringKey(Self.namePrompt), bundle: .module)) {
          Text(LocalizedStringKey(Self.name), bundle: .module)
        }
        FieldErrorText(key: viewModel.errors["name"])
        CurrencyPicker(titleKey: Self.currency, selection: $viewModel.currency)
        FieldErrorText(key: viewModel.errors["currency"])
      } footer: {
        Text(LocalizedStringKey(Self.currencyFooter), bundle: .module)
      }
      Section {
        TextField(text: $viewModel.displayName) {
          Text(LocalizedStringKey(Self.displayName), bundle: .module)
        }
        .textContentType(.name)
        FieldErrorText(key: viewModel.errors["display_name"])
      } footer: {
        Text(LocalizedStringKey(Self.displayNameFooter), bundle: .module)
      }
      Section {
        FormMessage(key: viewModel.errors.message)
        SubmitButton(
          titleKey: HomeView.createGroup, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit
        ) {
          if let group = await viewModel.submit() { onCreated(group) }
        }
      }
    }
    .navigationTitle(Text(LocalizedStringKey(HomeView.createGroup), bundle: .module))
    .toolbar {
      ToolbarItem(placement: .cancellationAction) {
        Button {
          dismiss()
        } label: {
          Text(LocalizedStringKey(Self.cancel), bundle: .module)
        }
      }
    }
  }

  nonisolated static let name = "Group name"
  nonisolated static let namePrompt = "Goa trip"
  nonisolated static let currency = "Group Currency"
  nonisolated static let currencyFooter =
    "Balances and Settlements are in this currency. It can't change once there are Expenses."
  nonisolated static let displayName = "Your name in this Group"
  nonisolated static let displayNameFooter = "Others in the Group see this name."
  nonisolated static let cancel = "Cancel"
  nonisolated static let allKeys = [name, namePrompt, currency, currencyFooter, displayName, displayNameFooter, cancel]
}
