import Domain
import SwiftUI

/// An Expense: amount (and, if paid in another currency, its Original Amount
/// and Exchange Rate), payer, Category, date, note and each Member's Share.
/// Its creator may Edit or Withdraw it (FR-E6).
struct ExpenseDetailView: View {
  @Bindable var viewModel: ExpenseDetailViewModel
  /// Called after an edit or withdrawal, so the Group reloads.
  var onChanged: () -> Void = {}
  @State private var confirmsWithdraw = false
  @State private var editing: AddExpenseViewModel?

  var body: some View {
    content
      .navigationTitle(Text(LocalizedStringKey(Self.title), bundle: .module))
      .task { await viewModel.load() }
      .toolbar {
        if viewModel.canEdit, let e = viewModel.state.value {
          ToolbarItem(placement: .primaryAction) {
            Button {
              editing = AddExpenseViewModel(editing: e, group: viewModel.group, repository: viewModel.repository)
            } label: {
              Text(LocalizedStringKey(Self.edit), bundle: .module)
            }
          }
        }
      }
      .sheet(item: $editing) { model in
        NavigationStack {
          AddExpenseView(viewModel: model) { saved in
            editing = nil
            viewModel.edited(saved)
            onChanged()
          }
        }
      }
      .confirmationDialog(
        Text(LocalizedStringKey(Self.withdrawTitle), bundle: .module), isPresented: $confirmsWithdraw,
        titleVisibility: .visible
      ) {
        Button(role: .destructive) {
          Task { if await viewModel.withdraw() { onChanged() } }
        } label: {
          Text(LocalizedStringKey(SettlementDetailView.withdraw), bundle: .module)
        }
      }
      .alert(
        Text(LocalizedStringKey(SettlementDetailView.withdrawFailed), bundle: .module),
        isPresented: Binding(get: { viewModel.actionError != nil }, set: { if !$0 { viewModel.dismissError() } })
      ) {
        Button {
          viewModel.dismissError()
        } label: {
          Text(LocalizedStringKey(CommonKeys.ok), bundle: .module)
        }
      } message: {
        Text(LocalizedStringKey(viewModel.actionError ?? ""), bundle: .module)
      }
  }

  @ViewBuilder private var content: some View {
    switch viewModel.state {
    case .loading:
      LoadingView()
    case .failed(let key):
      FailedView(messageKey: key) { await viewModel.load() }
    case .loaded(let e):
      List {
        Section {
          Text(verbatim: e.amount.formatted()).font(.largeTitle.monospacedDigit())
            .strikethrough(e.state == .withdrawn)
          if e.state == .withdrawn {
            Label {
              Text(LocalizedStringKey(SettlementDetailView.withdrawn), bundle: .module)
            } icon: {
              Image(systemName: "arrow.uturn.backward.circle").accessibilityHidden(true)
            }
          }
          if let rate = e.exchangeRate {
            LabeledContent {
              Text(verbatim: e.originalAmount.formatted()).monospacedDigit()
            } label: {
              Text(LocalizedStringKey(Self.originalAmount), bundle: .module)
            }
            LabeledContent {
              Text(
                "\(ExchangeRate.display(rate)) \(e.amount.currency) per 1 \(e.originalAmount.currency)", bundle: .module
              )
              .monospacedDigit()
            } label: {
              Text(LocalizedStringKey(Self.exchangeRate), bundle: .module)
            }
          }
          if let note = e.note { Text(verbatim: note) }
          CategoryLabel(category: e.category)
          LabeledContent {
            MemberName(name: viewModel.name(e.payerID))
          } label: {
            Text(LocalizedStringKey(AddExpenseView.paidBy), bundle: .module)
          }
          LabeledContent {
            Text(verbatim: ExpenseRow.dayText(e.spentOn))
          } label: {
            Text(LocalizedStringKey(AddExpenseView.date), bundle: .module)
          }
          LabeledContent {
            Text(LocalizedStringKey(AddExpenseView.methodName(e.splitMethod)), bundle: .module)
          } label: {
            Text(LocalizedStringKey(AddExpenseView.split), bundle: .module)
          }
        }
        Section {
          ForEach(e.shares, id: \.memberID) { share in
            LabeledContent {
              Text(verbatim: share.amount.formatted()).monospacedDigit()
            } label: {
              MemberName(name: viewModel.name(share.memberID))
              if let entry = Self.entryText(e.splitMethod, share.input) {
                entry
              }
            }
          }
        } header: {
          Text(LocalizedStringKey(Self.shares), bundle: .module)
        }
        if viewModel.canWithdraw {
          Section {
            Button(role: .destructive) {
              confirmsWithdraw = true
            } label: {
              Text(LocalizedStringKey(SettlementDetailView.withdraw), bundle: .module).frame(minHeight: 44)
            }
            .disabled(viewModel.isWithdrawing)
          } footer: {
            Text(LocalizedStringKey(SettlementDetailView.withdrawFooter), bundle: .module)
          }
        }
      }
    }
  }

  /// What was entered for a Member, under their name: "33.33%" or "Ratio
  /// part 2". An exact amount is the Share itself, so it shows nothing.
  static func entryText(_ method: SplitMethod, _ input: String?) -> Text? {
    guard let input else { return nil }
    switch method {
    case .percentage: return Text(verbatim: method.displayInput(input))
    case .ratio: return Text("Ratio part \(method.displayInput(input))", bundle: .module)
    case .equal, .exact: return nil
    }
  }

  nonisolated static let title = "Expense"
  nonisolated static let shares = "Shares"
  /// The catalog key `entryText` produces for a ratio part.
  nonisolated static let ratioPartFormat = "Ratio part %@"
  nonisolated static let originalAmount = "Original Amount"
  nonisolated static let exchangeRate = "Exchange Rate"
  /// The catalog key of the rate row: rate, Group Currency, Expense's
  /// currency.
  nonisolated static let rateFormat = "%@ %@ per 1 %@"
  nonisolated static let edit = "Edit"
  nonisolated static let withdrawTitle = "Withdraw this Expense?"
  nonisolated static let allKeys = [
    title, shares, ratioPartFormat, originalAmount, exchangeRate, rateFormat, MemberName.unknown, edit, withdrawTitle,
  ]
}

/// A Member's display name, or "Unknown Member" if the Group no longer
/// lists them.
struct MemberName: View {
  let name: String?

  var body: some View {
    if let name {
      Text(verbatim: name)
    } else {
      Text(LocalizedStringKey(Self.unknown), bundle: .module)
    }
  }

  nonisolated static let unknown = "Unknown Member"
}

/// An Expense in the Group's list.
struct ExpenseRow: View {
  let expense: ExpenseSummary
  let payerName: String?

  var body: some View {
    HStack(alignment: .firstTextBaseline) {
      Image(systemName: CategoryLabel.symbol(expense.category)).foregroundStyle(.secondary).accessibilityHidden(true)
      VStack(alignment: .leading, spacing: 2) {
        if let note = expense.note {
          Text(verbatim: note)
        } else {
          Text(LocalizedStringKey(CategoryLabel.nameKey(expense.category)), bundle: .module)
        }
        Text(
          LocalizedStringKey("\(payerName ?? Self.unknownName) paid · \(Self.dayText(expense.spentOn))"),
          bundle: .module
        )
        .font(.footnote).foregroundStyle(.secondary)
      }
      Spacer()
      VStack(alignment: .trailing, spacing: 2) {
        Text(verbatim: expense.amount.formatted()).monospacedDigit().strikethrough(expense.state == .withdrawn)
        if expense.state == .withdrawn {
          Text(LocalizedStringKey(SettlementDetailView.withdrawn), bundle: .module).font(.footnote)
            .foregroundStyle(.secondary)
        }
      }
    }
    .accessibilityElement(children: .combine)
  }

  /// The catalog key the subtitle's interpolation produces.
  nonisolated static let paidOnFormat = "%@ paid · %@"

  private static var unknownName: String {
    String(localized: String.LocalizationValue(MemberName.unknown), bundle: .module)
  }

  /// "2026-10-01" in the User's style, such as "1 Oct 2026".
  static func dayText(_ day: String) -> String {
    let parts = day.split(separator: "-").compactMap { Int($0) }
    guard parts.count == 3,
      let date = Calendar(identifier: .gregorian).date(
        from: DateComponents(year: parts[0], month: parts[1], day: parts[2]))
    else { return day }
    return date.formatted(date: .abbreviated, time: .omitted)
  }
}
