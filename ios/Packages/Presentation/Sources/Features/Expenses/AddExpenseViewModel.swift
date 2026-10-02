import Domain
import Foundation
import Observation

/// Add Expense (FR-E1–E3): amount in the Group Currency, payer, Category,
/// optional note, date, the Split method and the Members who share it, with
/// an entry each for an exact, percentage or ratio Split. The form checks
/// only that entries are well-formed; whether they add up is the server's
/// check (ADR-0006). The live preview shows the exact Shares from the server, the same
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
  /// How the Expense is divided. Changing it clears the entries, which mean
  /// something else under another method.
  public var method: SplitMethod = .equal {
    didSet { if method != oldValue { entryTexts = [:] } }
  }
  /// What the User typed for each Member: an amount, a percentage or a ratio
  /// part, depending on `method`.
  public var entryTexts: [UUID: String] = [:]
  public private(set) var isSubmitting = false
  private(set) var errors = FormErrors()
  /// The latest preview's Shares, or nil before one arrives.
  private(set) var preview: ExpensePreview?
  /// Why the latest preview failed (a catalog key), if it isn't shown at a
  /// field.
  private(set) var previewError: String?
  /// The latest preview's refused fields.
  private var previewFields = FormErrors()

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

  /// The Members sharing the Expense, in joining order: the Split's order.
  var sharingMembers: [Member] { members.filter { splitMembers.contains($0.id) } }

  /// The input as it stands, or nil while the amount isn't a valid amount,
  /// nobody shares it, or a sharing Member's entry is missing or malformed.
  var input: ExpenseInput? {
    guard let amount = Money.parse(amountText, currency: group.currency, locale: locale), !splitMembers.isEmpty
    else { return nil }
    var entries: [SplitEntry] = []
    for m in sharingMembers {
      guard method.takesInput else {
        entries.append(SplitEntry(memberID: m.id))
        continue
      }
      guard let value = entry(for: m.id) else { return nil }
      entries.append(SplitEntry(memberID: m.id, input: value))
    }
    let trimmed = note.trimmingCharacters(in: .whitespacesAndNewlines)
    return ExpenseInput(
      payerID: payerID, amount: amount, category: category, note: trimmed.isEmpty ? nil : trimmed,
      spentOn: Self.day(spentOn), method: method, members: entries)
  }

  private func entry(for memberID: UUID) -> String? {
    method.input(from: entryTexts[memberID] ?? "", currency: group.currency, locale: locale)
  }

  /// Whether a Member's typed entry is unreadable (blank isn't: it's not
  /// filled in yet).
  func entryIsInvalid(_ memberID: UUID) -> Bool {
    let text = (entryTexts[memberID] ?? "").trimmingCharacters(in: .whitespaces)
    return method.takesInput && !text.isEmpty && entry(for: memberID) == nil
  }

  /// The server's refusal of a sharing Member's entry, if any.
  func entryError(for memberID: UUID) -> String? {
    guard let i = sharingMembers.firstIndex(where: { $0.id == memberID }) else { return nil }
    return fieldError("split/members/\(i)/input")
  }

  /// The server's refusal of the Split as a whole, such as percentages that
  /// don't add up to 100.
  var splitError: String? { fieldError("split") }

  /// A refused field, from the last save or else the last preview.
  func fieldError(_ field: String) -> String? { errors[field] ?? previewFields[field] }

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
      previewFields = FormErrors()
      return
    }
    do {
      let p = try await repository.preview(groupID: group.id, input: input)
      // Keep only the answer for the input still on screen.
      if self.input == input {
        preview = p
        previewError = nil
        previewFields = FormErrors()
      }
    } catch {
      if self.input == input {
        preview = nil
        previewFields = FormErrors(error)
        previewError = Self.shownAtAField(error) ? nil : ServiceErrorMessage.key(for: error)
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

  /// Whether every refused field has a place on the form to show it.
  private static func shownAtAField(_ error: ServiceError) -> Bool {
    guard case .invalidFields(let issues) = error else { return false }
    let shown: Set<String> = ["amount/minor", "amount/currency", "note", "split", "split/members"]
    return issues.allSatisfy { shown.contains($0.field) || $0.field.hasSuffix("/input") }
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
