import Domain
import Foundation
import Observation

/// Record Settlement (FR-S1): who paid whom, how much in the Group
/// Currency, when, and an optional note. It can start from a Settle-up
/// Suggestion. An overpayment asks for confirmation, then saves (FR-S2).
@MainActor
@Observable
public final class RecordSettlementViewModel {
  public let group: Domain.Group
  public var fromMemberID: UUID
  public var toMemberID: UUID
  public var amountText: String
  public var settledOn: Date
  public var note = ""
  public private(set) var isSubmitting = false
  private(set) var errors = FormErrors()
  /// Set when the server asks to confirm an overpayment.
  private(set) var needsConfirmation = false

  private let repository: any SettlementsRepository
  private let locale: Locale
  private var keys = WriteKeys<RecordAttempt>()

  /// What makes a submission distinct: confirming is a new request with a
  /// new Idempotency-Key.
  private struct RecordAttempt: Hashable, Sendable {
    let input: SettlementInput
    let acknowledged: Bool
  }

  public init(
    group: Domain.Group, repository: any SettlementsRepository, suggestion: SettleUpSuggestion? = nil,
    today: Date = .now, locale: Locale = .current
  ) {
    self.group = group
    self.repository = repository
    self.locale = locale
    self.settledOn = today
    let members = group.activeMembers
    let me = group.myMemberID
    self.fromMemberID = suggestion?.fromMemberID ?? me
    self.toMemberID = suggestion?.toMemberID ?? (members.first { $0.id != me }?.id ?? me)
    self.amountText = suggestion?.amount.editableText(locale: locale) ?? ""
  }

  var members: [Member] { group.activeMembers }

  var input: SettlementInput? {
    guard let amount = Money.parse(amountText, currency: group.currency, locale: locale), fromMemberID != toMemberID
    else { return nil }
    let trimmed = note.trimmingCharacters(in: .whitespacesAndNewlines)
    return SettlementInput(
      fromMemberID: fromMemberID, toMemberID: toMemberID, amount: amount,
      settledOn: AddExpenseViewModel.day(settledOn), note: trimmed.isEmpty ? nil : trimmed)
  }

  public var canSubmit: Bool { input != nil && !isSubmitting }

  var sameMember: Bool { fromMemberID == toMemberID }

  /// Records the Settlement; returns it on success. `acknowledge` confirms
  /// an overpayment the server warned about.
  public func submit(acknowledge: Bool = false) async -> Settlement? {
    guard let input, !isSubmitting else { return nil }
    isSubmitting = true
    defer { isSubmitting = false }
    let key = keys.key(for: RecordAttempt(input: input, acknowledged: acknowledge))
    do {
      let s = try await repository.record(groupID: group.id, input: input, acknowledgeWarnings: acknowledge, key: key)
      keys.succeeded()
      errors = FormErrors()
      needsConfirmation = false
      return s
    } catch .needsConfirmation {
      needsConfirmation = true
      return nil
    } catch {
      errors = FormErrors(error)
      return nil
    }
  }

  func cancelConfirmation() { needsConfirmation = false }
}

extension RecordSettlementViewModel: Identifiable {
  public nonisolated var id: ObjectIdentifier { ObjectIdentifier(self) }
}
