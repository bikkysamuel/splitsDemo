import Domain
import Foundation
import Observation

/// Add Expense (FR-E1–E3, FR-E5), or Edit Expense for its creator: the
/// form pre-filled, saved as a new revision with the version it changes
/// (FR-E6). Amount in the Group Currency or, with an Exchange Rate, in
/// another currency, payer, Category, optional note, date, the Split method
/// and the Members who share it, with an entry each for an exact,
/// percentage or ratio Split. The form checks only that entries are
/// well-formed; whether they add up is the server's check. The live preview
/// shows the exact Shares from the server, the same calculation the save
/// uses (FR-E4, ADR-0006).
@MainActor
@Observable
public final class AddExpenseViewModel {
  public let group: Domain.Group
  public var amountText = ""
  /// The currency the money was paid in; the Group Currency to start.
  public var currency: String
  /// The Exchange Rate typed, used only while `currency` isn't the Group
  /// Currency: units of the Group Currency per one unit of `currency`.
  public var rateText = ""
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
  /// The Expense being edited, as last loaded; nil when adding one.
  public private(set) var editing: Expense?
  /// Whether the last save was refused because someone else changed the
  /// Expense meanwhile (NFR-R4): `reload()` fetches the current one.
  private(set) var isStale = false
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
  private var editKeys = WriteKeys<EditAttempt>()

  public init(group: Domain.Group, repository: any ExpensesRepository, today: Date = .now, locale: Locale = .current) {
    self.group = group
    self.repository = repository
    self.locale = locale
    self.spentOn = today
    self.currency = group.currency
    self.payerID = group.myMemberID
    self.splitMembers = Set(group.activeMembers.map(\.id))
  }

  /// Edit Expense: the form pre-filled with `expense` as saved.
  public convenience init(
    editing expense: Expense, group: Domain.Group, repository: any ExpensesRepository, locale: Locale = .current
  ) {
    self.init(group: group, repository: repository, locale: locale)
    fill(expense)
  }

  public var isEditing: Bool { editing != nil }

  /// Puts a saved Expense into the form, as text that reads back to it.
  private func fill(_ e: Expense) {
    editing = e
    currency = e.originalAmount.currency
    amountText = e.originalAmount.editableText(locale: locale)
    rateText = e.exchangeRate.map { ExchangeRate.editableText($0, locale: locale) } ?? ""
    payerID = e.payerID
    category = e.category
    note = e.note ?? ""
    spentOn = Self.date(e.spentOn) ?? spentOn
    method = e.splitMethod
    splitMembers = Set(e.shares.map(\.memberID))
    var entries: [UUID: String] = [:]
    for share in e.shares {
      if let input = share.input {
        entries[share.memberID] = e.splitMethod.editableInput(input, currency: group.currency, locale: locale)
      }
    }
    entryTexts = entries
  }

  /// Fetches the Expense as it now is into the form, after someone else
  /// changed it; what was typed is replaced.
  public func reload() async {
    guard let editing else { return }
    do {
      fill(try await repository.expense(id: editing.id))
      isStale = false
      errors = FormErrors()
      preview = nil
    } catch {
      errors = FormErrors(error)
    }
  }

  /// The Members who can pay or share: active ones, in joining order.
  var members: [Member] { group.activeMembers }

  /// The Members sharing the Expense, in joining order: the Split's order.
  var sharingMembers: [Member] { members.filter { splitMembers.contains($0.id) } }

  /// Whether the amount is in another currency than the Group Currency, so
  /// it needs an Exchange Rate.
  var needsRate: Bool { currency != group.currency }

  /// The typed rate in the server's form, or nil while it isn't readable.
  private var rate: String? { ExchangeRate.input(from: rateText, locale: locale) }

  /// Whether the typed rate is unreadable (blank isn't: it's not filled in
  /// yet).
  var rateIsInvalid: Bool {
    needsRate && !rateText.trimmingCharacters(in: .whitespaces).isEmpty && rate == nil
  }

  /// The amount in the Group Currency from the latest preview, while the
  /// Expense is in another currency (ADR-0006: the server converts).
  var convertedAmount: Money? { needsRate ? preview?.amount : nil }

  /// The input as it stands, or nil while the amount isn't a valid amount,
  /// another currency's rate is missing or malformed, nobody shares it, or a
  /// sharing Member's entry is missing or malformed.
  var input: ExpenseInput? {
    guard let amount = Money.parse(amountText, currency: currency, locale: locale), !splitMembers.isEmpty
    else { return nil }
    let exchangeRate = needsRate ? rate : nil
    if needsRate && exchangeRate == nil { return nil }
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
      payerID: payerID, amount: amount, exchangeRate: exchangeRate, category: category,
      note: trimmed.isEmpty ? nil : trimmed,
      spentOn: Self.day(spentOn), method: method, members: entries)
  }

  /// Exact entries are in the Group Currency whatever the Expense's
  /// currency (Q97): they divide the converted amount.
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

  /// A refused field, from the last save (until the next preview answers)
  /// or else the last preview.
  func fieldError(_ field: String) -> String? { errors[field] ?? previewFields[field] }

  public var canSubmit: Bool { input != nil && !isSubmitting }

  /// Whether the typed amount is unreadable (shown under the field).
  var amountIsInvalid: Bool {
    !amountText.trimmingCharacters(in: .whitespaces).isEmpty
      && Money.parse(amountText, currency: currency, locale: locale) == nil
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
        errors = FormErrors()
      }
    } catch {
      if self.input == input {
        // The preview speaks for the input now on screen; a refused save's
        // field errors may point at Split indexes that moved.
        errors = FormErrors()
        preview = nil
        previewFields = FormErrors(error)
        previewError = Self.shownAtAField(error) ? nil : ServiceErrorMessage.key(for: error)
      }
    }
  }

  /// Records the Expense, or saves the edit; returns it on success. A
  /// retry of the same input reuses its Idempotency-Key, so it is never
  /// saved twice.
  public func submit() async -> Expense? {
    guard let input, !isSubmitting else { return nil }
    isSubmitting = true
    defer { isSubmitting = false }
    do {
      let expense: Expense
      if let editing {
        expense = try await repository.editExpense(
          id: editing.id, version: editing.version, input: input,
          key: editKeys.key(for: EditAttempt(version: editing.version, input: input)))
        editKeys.succeeded()
      } else {
        expense = try await repository.createExpense(groupID: group.id, input: input, key: keys.key(for: input))
        keys.succeeded()
      }
      errors = FormErrors()
      return expense
    } catch {
      errors = FormErrors(error)
      isStale = error == .problem(.versionConflict)
      return nil
    }
  }

  /// One edit request: the same input on another version is another
  /// request, so it takes another Idempotency-Key.
  private struct EditAttempt: Hashable {
    let version: Int
    let input: ExpenseInput
  }

  /// Whether every refused field has a place on the form to show it.
  private static func shownAtAField(_ error: ServiceError) -> Bool {
    guard case .invalidFields(let issues) = error else { return false }
    let shown: Set<String> = ["amount/minor", "amount/currency", "exchange_rate", "note", "split", "split/members"]
    return issues.allSatisfy { shown.contains($0.field) || $0.field.hasSuffix("/input") }
  }

  /// The day "yyyy-MM-dd" at the start of that day in the User's calendar.
  static func date(_ day: String, calendar: Calendar = .current) -> Date? {
    let parts = day.split(separator: "-").compactMap { Int($0) }
    guard parts.count == 3 else { return nil }
    return calendar.date(from: DateComponents(year: parts[0], month: parts[1], day: parts[2]))
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
