import Domain
import SwiftUI

/// A Group's Balances and Settle-up Suggestions (FR-B1, FR-B2). Owed and
/// owing are told apart by words and a symbol, never by colour alone, and
/// VoiceOver reads the sentence (NFR-A).
struct BalancesSection: View {
  let group: Domain.Group
  let balances: GroupBalances

  var body: some View {
    Section {
      ForEach(balances.balances, id: \.memberID) { b in
        HStack {
          Image(systemName: Self.symbol(b.direction)).accessibilityHidden(true)
          Text(verbatim: Self.sentence(b, name: name(b.memberID)))
        }
        .accessibilityElement(children: .combine)
      }
    } header: {
      Text(LocalizedStringKey(Self.balancesHeader), bundle: .module)
    }
    Section {
      if balances.suggestions.isEmpty {
        Text(LocalizedStringKey(Self.allSettled), bundle: .module).foregroundStyle(.secondary)
      }
      ForEach(balances.suggestions, id: \.self) { s in
        Text(verbatim: Self.suggestion(s, from: name(s.fromMemberID), to: name(s.toMemberID)))
      }
    } header: {
      Text(LocalizedStringKey(Self.suggestionsHeader), bundle: .module)
    }
  }

  private func name(_ id: UUID) -> String {
    group.member(id)?.displayName ?? String(localized: String.LocalizationValue(MemberName.unknown), bundle: .module)
  }

  static func sentence(_ b: MemberBalance, name: String) -> String {
    let amount = b.balance.formatted(signed: false)
    switch b.direction {
    case .owed: return format(Self.owedFormat, name, amount)
    case .owes: return format(Self.owesFormat, name, amount)
    case .settled: return format(Self.settledFormat, name)
    }
  }

  static func suggestion(_ s: SettleUpSuggestion, from: String, to: String) -> String {
    format(Self.paysFormat, from, to, s.amount.formatted())
  }

  static func symbol(_ d: MemberBalance.Direction) -> String {
    switch d {
    case .owed: "arrow.down.circle"
    case .owes: "arrow.up.circle"
    case .settled: "checkmark.circle"
    }
  }

  private static func format(_ key: String, _ args: CVarArg...) -> String {
    String(format: String(localized: String.LocalizationValue(key), bundle: .module), arguments: args)
  }

  nonisolated static let balancesHeader = "Balances"
  nonisolated static let suggestionsHeader = "Settle up"
  nonisolated static let allSettled = "Everyone is settled up."
  nonisolated static let owedFormat = "%1$@ is owed %2$@"
  nonisolated static let owesFormat = "%1$@ owes %2$@"
  nonisolated static let settledFormat = "%@ is settled up"
  nonisolated static let paysFormat = "%1$@ pays %2$@ %3$@"
  nonisolated static let allKeys = [
    balancesHeader, suggestionsHeader, allSettled, owedFormat, owesFormat, settledFormat, paysFormat,
  ]
}
