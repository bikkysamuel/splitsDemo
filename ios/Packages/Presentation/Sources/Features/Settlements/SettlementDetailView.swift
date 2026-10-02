import Domain
import SwiftUI

/// A Settlement, with Withdraw for its creator.
struct SettlementDetailView: View {
  @Bindable var viewModel: SettlementDetailViewModel
  let onChanged: () -> Void
  @State private var confirmsWithdraw = false

  var body: some View {
    content
      .navigationTitle(Text(LocalizedStringKey(Self.title), bundle: .module))
      .task { await viewModel.load() }
      .confirmationDialog(
        Text(LocalizedStringKey(Self.withdrawTitle), bundle: .module), isPresented: $confirmsWithdraw,
        titleVisibility: .visible
      ) {
        Button(role: .destructive) {
          Task { if await viewModel.withdraw() { onChanged() } }
        } label: {
          Text(LocalizedStringKey(Self.withdraw), bundle: .module)
        }
      }
      .alert(
        Text(LocalizedStringKey(GroupView.renameFailed), bundle: .module),
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
    case .loaded(let s):
      List {
        Section {
          Text(verbatim: s.amount.formatted()).font(.largeTitle.monospacedDigit())
          LabeledContent {
            MemberName(name: viewModel.name(s.fromMemberID))
          } label: {
            Text(LocalizedStringKey(RecordSettlementView.from), bundle: .module)
          }
          LabeledContent {
            MemberName(name: viewModel.name(s.toMemberID))
          } label: {
            Text(LocalizedStringKey(RecordSettlementView.to), bundle: .module)
          }
          LabeledContent {
            Text(verbatim: ExpenseRow.dayText(s.settledOn))
          } label: {
            Text(LocalizedStringKey(AddExpenseView.date), bundle: .module)
          }
          if let note = s.note { Text(verbatim: note) }
          if s.state == .withdrawn {
            Label {
              Text(LocalizedStringKey(Self.withdrawn), bundle: .module)
            } icon: {
              Image(systemName: "arrow.uturn.backward.circle").accessibilityHidden(true)
            }
          }
        }
        if viewModel.canWithdraw {
          Section {
            Button(role: .destructive) {
              confirmsWithdraw = true
            } label: {
              Text(LocalizedStringKey(Self.withdraw), bundle: .module).frame(minHeight: 44)
            }
            .disabled(viewModel.isWithdrawing)
          } footer: {
            Text(LocalizedStringKey(Self.withdrawFooter), bundle: .module)
          }
        }
      }
    }
  }

  nonisolated static let title = "Settlement"
  nonisolated static let withdraw = "Withdraw"
  nonisolated static let withdrawTitle = "Withdraw this Settlement?"
  nonisolated static let withdrawFooter = "It stays in the history but no longer counts toward Balances."
  nonisolated static let withdrawn = "Withdrawn"
  nonisolated static let allKeys = [title, withdraw, withdrawTitle, withdrawFooter, withdrawn]
}

/// A Settlement in the Group's list.
struct SettlementRow: View {
  let settlement: Settlement
  let fromName: String?
  let toName: String?

  var body: some View {
    HStack(alignment: .firstTextBaseline) {
      Image(systemName: "arrow.right.circle").foregroundStyle(.secondary).accessibilityHidden(true)
      VStack(alignment: .leading, spacing: 2) {
        Text(verbatim: Self.line(settlement, from: fromName, to: toName))
          .strikethrough(settlement.state == .withdrawn)
        Text(verbatim: ExpenseRow.dayText(settlement.settledOn)).font(.footnote).foregroundStyle(.secondary)
      }
      Spacer()
      if settlement.state == .withdrawn {
        Text(LocalizedStringKey(SettlementDetailView.withdrawn), bundle: .module).font(.footnote)
          .foregroundStyle(.secondary)
      }
    }
    .accessibilityElement(children: .combine)
  }

  static func line(_ s: Settlement, from: String?, to: String?) -> String {
    let unknown = String(localized: String.LocalizationValue(MemberName.unknown), bundle: .module)
    return String(
      format: String(localized: String.LocalizationValue(Self.paidFormat), bundle: .module), from ?? unknown,
      to ?? unknown, s.amount.formatted())
  }

  nonisolated static let paidFormat = "%1$@ paid %2$@ %3$@"
}
