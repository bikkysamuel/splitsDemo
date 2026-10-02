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
        .accessibilityHint(Text(verbatim: CurrencyPicker.label(viewModel.group.currency)))
        if viewModel.amountIsInvalid {
          FieldErrorText(key: Self.amountInvalid)
        }
        FieldErrorText(key: viewModel.fieldError("amount/minor") ?? viewModel.fieldError("amount/currency"))
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
        FieldErrorText(key: viewModel.fieldError("note"))
      } header: {
        Text(verbatim: CurrencyPicker.label(viewModel.group.currency))
      }
      Section {
        Picker(selection: $viewModel.method) {
          ForEach(SplitMethod.allCases, id: \.self) { method in
            Text(LocalizedStringKey(Self.methodName(method)), bundle: .module).tag(method)
          }
        } label: {
          Text(LocalizedStringKey(Self.split), bundle: .module)
        }
        ForEach(viewModel.members) { m in
          SplitMemberRow(viewModel: viewModel, member: m)
        }
        FieldErrorText(key: viewModel.fieldError("split/members"))
      } header: {
        Text(LocalizedStringKey(Self.sharedBy), bundle: .module)
      } footer: {
        if let key = viewModel.splitError ?? viewModel.previewError {
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
  nonisolated static let split = "Split"
  nonisolated static let sharedBy = "Shared by"
  nonisolated static let previewFooter = "Shares from the server, exactly as they'll be saved."
  nonisolated static let save = "Save Expense"
  nonisolated static let allKeys =
    [title, amount, amountInvalid, paidBy, category, date, note, split, sharedBy, previewFooter, save]
    + SplitMethod.allCases.map(methodName) + SplitMemberRow.allKeys

  /// The catalog key naming a Split method.
  nonisolated static func methodName(_ method: SplitMethod) -> String {
    switch method {
    case .equal: "Equally"
    case .exact: "By exact amounts"
    case .percentage: "By percentages"
    case .ratio: "By ratio"
    }
  }
}

/// One Member in the Split: whether they share it, their entry for an
/// exact, percentage or ratio Split, and their Share from the preview.
private struct SplitMemberRow: View {
  @Bindable var viewModel: AddExpenseViewModel
  let member: Member

  private var isSharing: Bool { viewModel.splitMembers.contains(member.id) }
  private var method: SplitMethod { viewModel.method }

  var body: some View {
    VStack(alignment: .leading, spacing: 4) {
      HStack {
        Button {
          viewModel.toggle(member.id)
        } label: {
          HStack {
            Image(systemName: isSharing ? "checkmark.circle.fill" : "circle").accessibilityHidden(true)
            Text(verbatim: member.displayName)
          }
          .frame(minHeight: 44)
        }
        .buttonStyle(.borderless)
        .foregroundStyle(.primary)
        .accessibilityAddTraits(isSharing ? .isSelected : [])
        Spacer()
        if method.takesInput && isSharing {
          TextField(text: entry, prompt: Text(verbatim: method == .ratio ? "1" : "0")) {
            Text(Self.entryLabel(method, name: member.displayName), bundle: .module)
          }
          .multilineTextAlignment(.trailing)
          .monospacedDigit()
          .decimalEntry()
          .frame(maxWidth: 120)
          if method == .percentage {
            Text(verbatim: "%").foregroundStyle(.secondary).accessibilityHidden(true)
          }
        }
        if let share = viewModel.preview?.shares.first(where: { $0.memberID == member.id }) {
          Text(verbatim: share.amount.formatted()).monospacedDigit().foregroundStyle(.secondary)
            .accessibilityLabel(Text(LocalizedStringKey("Share \(share.amount.formatted())"), bundle: .module))
        }
      }
      if viewModel.entryIsInvalid(member.id) {
        FieldErrorText(key: Self.entryInvalid(method))
      }
      FieldErrorText(key: viewModel.entryError(for: member.id))
    }
  }

  private var entry: Binding<String> {
    Binding(
      get: { viewModel.entryTexts[member.id] ?? "" },
      set: { viewModel.entryTexts[member.id] = $0 })
  }

  /// The entry field's label for VoiceOver, such as "Percentage for Bob";
  /// the interpolations produce the `…For` catalog keys.
  static func entryLabel(_ method: SplitMethod, name: String) -> LocalizedStringKey {
    switch method {
    case .percentage: "Percentage for \(name)"
    case .ratio: "Ratio part for \(name)"
    default: "Amount for \(name)"
    }
  }

  /// Why a typed entry can't be read, for the method.
  static func entryInvalid(_ method: SplitMethod) -> String {
    switch method {
    case .percentage: percentageInvalid
    case .ratio: ratioInvalid
    default: AddExpenseView.amountInvalid
    }
  }

  // Catalog keys of the interpolated strings above and in `body`.
  nonisolated static let amountFor = "Amount for %@"
  nonisolated static let percentageFor = "Percentage for %@"
  nonisolated static let ratioPartFor = "Ratio part for %@"
  nonisolated static let shareFormat = "Share %@"
  nonisolated static let percentageInvalid = "Enter a percentage with at most 2 decimal places, like 33.33."
  nonisolated static let ratioInvalid = "Enter a whole number, like 2."
  nonisolated static let allKeys = [
    amountFor, percentageFor, ratioPartFor, shareFormat, percentageInvalid, ratioInvalid,
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
