import Domain
import SwiftUI

/// An Expense: amount, payer, Category, date, note and each Member's Share.
struct ExpenseDetailView: View {
  @Bindable var viewModel: ExpenseDetailViewModel

  var body: some View {
    content
      .navigationTitle(Text(LocalizedStringKey(Self.title), bundle: .module))
      .task { await viewModel.load() }
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
        }
        Section {
          ForEach(e.shares, id: \.memberID) { share in
            LabeledContent {
              Text(verbatim: share.amount.formatted()).monospacedDigit()
            } label: {
              MemberName(name: viewModel.name(share.memberID))
            }
          }
        } header: {
          Text(LocalizedStringKey(Self.shares), bundle: .module)
        }
      }
    }
  }

  nonisolated static let title = "Expense"
  nonisolated static let shares = "Shares"
  nonisolated static let allKeys = [title, shares, MemberName.unknown]
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
      Text(verbatim: expense.amount.formatted()).monospacedDigit()
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
