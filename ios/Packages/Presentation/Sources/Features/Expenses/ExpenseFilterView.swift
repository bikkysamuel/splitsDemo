import Domain
import SwiftUI

/// What the Expense filter sheet edits: an `ExpenseFilter`, with its
/// dates as `Date`s and a switch each.
struct ExpenseFilterForm: Equatable {
  var memberID: UUID?
  var category: Domain.Category?
  var state: ExpenseState?
  var hasFrom: Bool
  var from: Date
  var hasTo: Bool
  var to: Date

  init(_ filter: ExpenseFilter, today: Date = .now, calendar: Calendar = .current) {
    memberID = filter.memberID
    category = filter.category
    state = filter.state
    hasFrom = filter.from != nil
    from = filter.from.flatMap { AddExpenseViewModel.date($0, calendar: calendar) } ?? today
    hasTo = filter.to != nil
    to = filter.to.flatMap { AddExpenseViewModel.date($0, calendar: calendar) } ?? today
  }

  /// The filter, with the last day moved up to the first if it was before
  /// it (the server refuses a reversed range).
  func filter(calendar: Calendar = .current) -> ExpenseFilter {
    let first = hasFrom ? AddExpenseViewModel.day(from, calendar: calendar) : nil
    var last = hasTo ? AddExpenseViewModel.day(to, calendar: calendar) : nil
    if let first, let l = last, l < first { last = first }
    return ExpenseFilter(memberID: memberID, category: category, from: first, to: last, state: state)
  }
}

/// Filters the Group's Expenses by Member, Category, state and dates
/// (FR-E8).
struct ExpenseFilterView: View {
  let group: Domain.Group
  let onApply: (ExpenseFilter) -> Void
  @State private var form: ExpenseFilterForm
  @Environment(\.dismiss) private var dismiss

  init(group: Domain.Group, filter: ExpenseFilter, onApply: @escaping (ExpenseFilter) -> Void) {
    self.group = group
    self.onApply = onApply
    _form = State(initialValue: ExpenseFilterForm(filter))
  }

  /// The states a filter offers: in M1 an Expense is accepted or withdrawn
  /// (D9).
  static let states: [ExpenseState] = [.accepted, .withdrawn]

  var body: some View {
    Form {
      Section {
        Picker(selection: $form.memberID) {
          Text(LocalizedStringKey(Self.any), bundle: .module).tag(UUID?.none)
          ForEach(group.members) { m in
            Text(verbatim: m.displayName).tag(UUID?.some(m.id))
          }
        } label: {
          Text(LocalizedStringKey(Self.member), bundle: .module)
        }
        Picker(selection: $form.category) {
          Text(LocalizedStringKey(Self.any), bundle: .module).tag(Domain.Category?.none)
          ForEach(Domain.Category.allCases, id: \.self) { c in
            CategoryLabel(category: c).tag(Domain.Category?.some(c))
          }
        } label: {
          Text(LocalizedStringKey(AddExpenseView.category), bundle: .module)
        }
        Picker(selection: $form.state) {
          Text(LocalizedStringKey(Self.any), bundle: .module).tag(ExpenseState?.none)
          ForEach(Self.states, id: \.self) { s in
            Text(LocalizedStringKey(Self.stateName(s)), bundle: .module).tag(ExpenseState?.some(s))
          }
        } label: {
          Text(LocalizedStringKey(Self.state), bundle: .module)
        }
      } footer: {
        Text(LocalizedStringKey(Self.memberFooter), bundle: .module)
      }
      Section {
        Toggle(isOn: $form.hasFrom) { Text(LocalizedStringKey(Self.from), bundle: .module) }
        if form.hasFrom {
          DatePicker(selection: $form.from, displayedComponents: .date) {
            Text(LocalizedStringKey(Self.from), bundle: .module)
          }
        }
        Toggle(isOn: $form.hasTo) { Text(LocalizedStringKey(Self.to), bundle: .module) }
        if form.hasTo {
          DatePicker(
            selection: $form.to, in: (form.hasFrom ? form.from : .distantPast)...,
            displayedComponents: .date
          ) {
            Text(LocalizedStringKey(Self.to), bundle: .module)
          }
        }
      } header: {
        Text(LocalizedStringKey(AddExpenseView.date), bundle: .module)
      }
      Section {
        Button(role: .destructive) {
          form = ExpenseFilterForm(.all)
        } label: {
          Text(LocalizedStringKey(Self.clear), bundle: .module).frame(minHeight: 44)
        }
        .disabled(!form.filter().isActive)
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
      ToolbarItem(placement: .confirmationAction) {
        Button {
          onApply(form.filter())
        } label: {
          Text(LocalizedStringKey(Self.apply), bundle: .module)
        }
      }
    }
  }

  nonisolated static let title = "Filter Expenses"
  nonisolated static let any = "Any"
  nonisolated static let member = "Member"
  nonisolated static let memberFooter = "Shows the Expenses this Member paid or shares."
  nonisolated static let state = "State"
  nonisolated static let accepted = "Accepted"
  nonisolated static let from = "From"
  nonisolated static let to = "To"
  nonisolated static let clear = "Clear Filters"
  nonisolated static let apply = "Apply"
  nonisolated static let allKeys = [title, any, member, memberFooter, state, accepted, from, to, clear, apply]

  /// The catalog key naming a state the filter offers.
  nonisolated static func stateName(_ s: ExpenseState) -> String {
    s == .withdrawn ? SettlementDetailView.withdrawn : accepted
  }
}
