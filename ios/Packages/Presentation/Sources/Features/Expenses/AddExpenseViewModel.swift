import Domain
import Foundation
import Observation

/// Add Expense (FR-E1, FR-E3): amount in the Group Currency, payer,
/// Category, optional note, date, and the Members to split equally among.
/// The live preview shows the exact Shares from the server, the same
/// calculation the save uses (FR-E4, ADR-0006).
@MainActor
@Observable
public final class AddExpenseViewModel {
  public let group: Domain.Group
  public var amountText = ""
  public var payerID: UUID
  public var category: Domain.Category = .foodDrink
  public var note = ""
  public var spentOn: Date
  public var splitMembers: Set<UUID>
  public private(set) var isSubmitting = false
  private(set) var errors = FormErrors()
  /// The latest preview's Shares, or nil before one arrives.
  private(set) var preview: ExpensePreview?
  /// Why the latest preview failed (a catalog key), if it did.
  private(set) var previewError: String?

  private let repository: any ExpensesRepository
  private let locale: Locale
  private var keys = WriteKeys<ExpenseInput>()

  public init(group: Domain.Group, repository: any ExpensesRepository, today: Date = .now, locale: Locale = .current) {
    self.group = group
    self.repository = repository
    self.locale = locale
    self.spentOn = today
    self.payerID = group.myMemberID
    self.splitMembers = Set(group.activeMembers.map(\.id))
  }

  /// The Members who can pay or share: active ones, in joining order.
  var members: [Member] { group.activeMembers }

  /// The input as it stands, or nil while the amount isn't a valid amount
  /// or nobody shares it.
  var input: ExpenseInput? {
    guard let amount = Money.parse(amountText, currency: group.currency, locale: locale), !splitMembers.isEmpty
    else { return nil }
    let trimmed = note.trimmingCharacters(in: .whitespacesAndNewlines)
    return ExpenseInput(
      payerID: payerID, amount: amount, category: category, note: trimmed.isEmpty ? nil : trimmed,
      spentOn: Self.day(spentOn), members: members.map(\.id).filter(splitMembers.contains))
  }

  public var canSubmit: Bool { input != nil && !isSubmitting }

  /// Whether the typed amount is unreadable (shown under the field).
  var amountIsInvalid: Bool {
    !amountText.trimmingCharacters(in: .whitespaces).isEmpty
      && Money.parse(amountText, currency: group.currency, locale: locale) == nil
  }

  func toggle(_ memberID: UUID) {
    if splitMembers.contains(memberID) { splitMembers.remove(memberID) } else { splitMembers.insert(memberID) }
  }

  /// Asks the server for the Shares of the current input. The view calls it
  /// after a short pause in typing.
  public func refreshPreview() async {
    guard let input else {
      preview = nil
      previewError = nil
      return
    }
    do {
      let p = try await repository.preview(groupID: group.id, input: input)
      // Keep only the answer for the input still on screen.
      if self.input == input {
        preview = p
        previewError = nil
      }
    } catch {
      if self.input == input {
        preview = nil
        previewError = ServiceErrorMessage.key(for: error)
      }
    }
  }

  /// Records the Expense; returns it on success. A retry of the same input
  /// reuses its Idempotency-Key, so it is never recorded twice.
  public func submit() async -> Expense? {
    guard let input, !isSubmitting else { return nil }
    isSubmitting = true
    defer { isSubmitting = false }
    do {
      let expense = try await repository.createExpense(groupID: group.id, input: input, key: keys.key(for: input))
      keys.succeeded()
      errors = FormErrors()
      return expense
    } catch {
      errors = FormErrors(error)
      return nil
    }
  }

  /// "yyyy-MM-dd" of the day in the User's calendar.
  static func day(_ date: Date, calendar: Calendar = .current) -> String {
    let c = calendar.dateComponents([.year, .month, .day], from: date)
    return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
  }
}

extension AddExpenseViewModel: Identifiable {
  public nonisolated var id: ObjectIdentifier { ObjectIdentifier(self) }
}

extension Domain.Group {
  /// Members who can pay or share an Expense.
  public var activeMembers: [Member] { members.filter { $0.status == .active } }

  public func member(_ id: UUID) -> Member? { members.first { $0.id == id } }
}
