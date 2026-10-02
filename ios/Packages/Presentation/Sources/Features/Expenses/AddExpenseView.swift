import Domain
import SwiftUI

/// The Add Expense form with its live split preview (FR-E1, FR-E4).
struct AddExpenseView: View {
  @Bindable var viewModel: AddExpenseViewModel
  let onAdded: (Expense) -> Void
  @Environment(\.dismiss) private var dismiss

  var body: some View {
    Form {
      Section {
        TextField(text: $viewModel.amountText, prompt: Text(verbatim: "0")) {
          Text(LocalizedStringKey(Self.amount), bundle: .module)
        }
        .font(.title2.monospacedDigit())
        .decimalEntry()
        .accessibilityHint(Text(verbatim: viewModel.group.currency))
        if viewModel.amountIsInvalid {
          FieldErrorText(key: Self.amountInvalid)
        }
        FieldErrorText(key: viewModel.errors["amount/minor"] ?? viewModel.errors["amount/currency"])
        Picker(selection: $viewModel.payerID) {
          ForEach(viewModel.members) { m in Text(verbatim: m.displayName).tag(m.id) }
        } label: {
          Text(LocalizedStringKey(Self.paidBy), bundle: .module)
        }
        Picker(selection: $viewModel.category) {
          ForEach(Domain.Category.allCases, id: \.self) { c in CategoryLabel(category: c).tag(c) }
        } label: {
          Text(LocalizedStringKey(Self.category), bundle: .module)
        }
        DatePicker(selection: $viewModel.spentOn, displayedComponents: .date) {
          Text(LocalizedStringKey(Self.date), bundle: .module)
        }
        TextField(text: $viewModel.note, axis: .vertical) {
          Text(LocalizedStringKey(Self.note), bundle: .module)
        }
        FieldErrorText(key: viewModel.errors["note"])
      } header: {
        Text(verbatim: CurrencyPicker.label(viewModel.group.currency))
      }
      Section {
        ForEach(viewModel.members) { m in
          Button {
            viewModel.toggle(m.id)
          } label: {
            HStack {
              Image(systemName: viewModel.splitMembers.contains(m.id) ? "checkmark.circle.fill" : "circle")
                .accessibilityHidden(true)
              Text(verbatim: m.displayName)
              Spacer()
              if let share = viewModel.preview?.shares.first(where: { $0.memberID == m.id }) {
                Text(verbatim: share.amount.formatted()).monospacedDigit().foregroundStyle(.secondary)
              }
            }
            .frame(minHeight: 44)
          }
          .foregroundStyle(.primary)
          .accessibilityAddTraits(viewModel.splitMembers.contains(m.id) ? .isSelected : [])
        }
        FieldErrorText(key: viewModel.errors["split/members"])
      } header: {
        Text(LocalizedStringKey(Self.splitEqually), bundle: .module)
      } footer: {
        if let key = viewModel.previewError {
          Text(LocalizedStringKey(key), bundle: .module)
        } else if viewModel.preview != nil {
          Text(LocalizedStringKey(Self.previewFooter), bundle: .module)
        }
      }
      Section {
        FormMessage(key: viewModel.errors.message)
        SubmitButton(titleKey: Self.save, isSubmitting: viewModel.isSubmitting, isEnabled: viewModel.canSubmit) {
          if let expense = await viewModel.submit() { onAdded(expense) }
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
    // Live preview: wait for a pause in typing, then ask the server.
    .task(id: viewModel.input) {
      try? await Task.sleep(for: .milliseconds(300))
      guard !Task.isCancelled else { return }
      await viewModel.refreshPreview()
    }
  }

  nonisolated static let title = "Add Expense"
  nonisolated static let amount = "Amount"
  nonisolated static let amountInvalid = "Enter an amount, like 250 or 99.50."
  nonisolated static let paidBy = "Paid by"
  nonisolated static let category = "Category"
  nonisolated static let date = "Date"
  nonisolated static let note = "Note (optional)"
  nonisolated static let splitEqually = "Split equally between"
  nonisolated static let previewFooter = "Shares from the server, exactly as they'll be saved."
  nonisolated static let save = "Save Expense"
  nonisolated static let allKeys = [
    title, amount, amountInvalid, paidBy, category, date, note, splitEqually, previewFooter, save,
  ]
}

extension View {
  fileprivate func decimalEntry() -> some View {
    #if os(iOS)
      self.keyboardType(.decimalPad)
    #else
      self
    #endif
  }
}
