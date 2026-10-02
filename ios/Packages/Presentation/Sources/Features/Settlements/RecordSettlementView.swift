import Domain
import SwiftUI

/// The Record Settlement form (FR-S1), with the overpayment confirmation
/// (FR-S2).
struct RecordSettlementView: View {
  @Bindable var viewModel: RecordSettlementViewModel
  let onRecorded: (Settlement) -> Void
  @Environment(\.dismiss) private var dismiss

  var body: some View {
    Form {
      Section {
        Picker(selection: $viewModel.fromMemberID) {
          ForEach(viewModel.members) { m in Text(verbatim: m.displayName).tag(m.id) }
        } label: {
          Text(LocalizedStringKey(Self.from), bundle: .module)
        }
        FieldErrorText(key: viewModel.errors["from_member_id"])
        Picker(selection: $viewModel.toMemberID) {
          ForEach(viewModel.members) { m in Text(verbatim: m.displayName).tag(m.id) }
        } label: {
          Text(LocalizedStringKey(Self.to), bundle: .module)
        }
        if viewModel.sameMember {
          FieldErrorText(key: ServiceErrorMessage.key(for: FieldIssue(field: "to_member_id", reason: .sameMember)))
        }
        FieldErrorText(key: viewModel.errors["to_member_id"])
        TextField(text: $viewModel.amountText, prompt: Text(verbatim: "0")) {
          Text(LocalizedStringKey(AddExpenseView.amount), bundle: .module)
        }
        .font(.title2.monospacedDigit())
        .settlementAmountEntry()
        .accessibilityHint(Text(verbatim: CurrencyPicker.label(viewModel.group.currency)))
        FieldErrorText(key: viewModel.errors["amount/minor"] ?? viewModel.errors["amount/currency"])
        DatePicker(selection: $viewModel.settledOn, displayedComponents: .date) {
          Text(LocalizedStringKey(AddExpenseView.date), bundle: .module)
        }
        TextField(text: $viewModel.note, axis: .vertical) {
          Text(LocalizedStringKey(AddExpenseView.note), bundle: .module)
        }
        FieldErrorText(key: viewModel.errors["note"])
      } header: {
        Text(verbatim: CurrencyPicker.label(viewModel.group.currency))
      } footer: {
        Text(LocalizedStringKey(Self.footer), bundle: .module)
      }
      Section {
        FormMessage(key: viewModel.errors.message)
        SubmitButton(titleKey: Self.save, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit) {
          if let s = await viewModel.submit() { onRecorded(s) }
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
    .alert(
      Text(LocalizedStringKey(Self.overpayTitle), bundle: .module),
      isPresented: Binding(get: { viewModel.needsConfirmation }, set: { if !$0 { viewModel.cancelConfirmation() } })
    ) {
      Button {
        Task { if let s = await viewModel.submit(acknowledge: true) { onRecorded(s) } }
      } label: {
        Text(LocalizedStringKey(Self.saveAnyway), bundle: .module)
      }
      Button(role: .cancel) {
        viewModel.cancelConfirmation()
      } label: {
        Text(LocalizedStringKey(CommonKeys.cancel), bundle: .module)
      }
    } message: {
      Text(LocalizedStringKey(Self.overpayMessage), bundle: .module)
    }
  }

  nonisolated static let title = "Record Settlement"
  nonisolated static let from = "Paid by"
  nonisolated static let to = "Paid to"
  nonisolated static let footer =
    "Record money paid outside the app. Any amount is fine, including part of what's owed."
  nonisolated static let save = "Save Settlement"
  nonisolated static let overpayTitle = "More than is owed"
  nonisolated static let overpayMessage =
    "This pays more than the payer owes or the receiver is owed. Save it anyway?"
  nonisolated static let saveAnyway = "Save anyway"
  nonisolated static let allKeys = [title, from, to, footer, save, overpayTitle, overpayMessage, saveAnyway]
}

extension View {
  fileprivate func settlementAmountEntry() -> some View {
    #if os(iOS)
      self.keyboardType(.decimalPad)
    #else
      self
    #endif
  }
}
