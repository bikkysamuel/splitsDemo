import APIClient
import Domain
import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import Data

/// Groups over the generated client, with JSON recorded from the server.
struct GroupsAPIRepositoryTests {
  @Test func createGroupSendsTheFieldsAndMapsTheGroup() async throws {
    let transport = try PathTransport(["POST /v1/groups": .fixture("group-create-201", status: .created)])
    let repository = try makeRepository(transport)

    let key = WriteKey()
    let group = try await repository.createGroup(name: "Goa trip", currency: "INR", displayName: "Alice", key: key)

    #expect(group.name == "Goa trip")
    #expect(group.currency == "INR")
    #expect(group.state == .active)
    #expect(group.version == 1)
    #expect(group.members.count == 1)
    #expect(group.myMember?.displayName == "Alice")
    #expect(group.isAdmin)
    let sent = try #require(await transport.sent.first)
    #expect(sent.request.headerFields[.authorization] == "Bearer saved-access")
    #expect(sent.request.headerFields[HTTPField.Name("Idempotency-Key")!] == key.value.uuidString)
    let body = try JSONSerialization.jsonObject(with: sent.body) as? [String: String]
    #expect(body == ["name": "Goa trip", "currency": "INR", "display_name": "Alice"])
  }

  @Test func groupsFollowsEveryPage() async throws {
    let transport = try PathTransport([
      "GET /v1/groups?limit=200": .fixture("groups-page1"),
      "GET /v1/groups?cursor=AaD6r9HWe5KT_ka1Hpa6Qw&limit=200": .fixture("groups-page2"),
    ])
    let repository = try makeRepository(transport)

    let groups = try await repository.groups()

    #expect(groups.map(\.name) == ["Goa trip", "Flat"])
    #expect(groups.map(\.currency) == ["INR", "EUR"])
  }

  @Test func refusedFieldsAreInvalidFields() async throws {
    let transport = try PathTransport([
      "POST /v1/groups": .fixture("group-create-400-validation-failed", status: .badRequest, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(
      throws: ServiceError.invalidFields([
        FieldIssue(field: "name", reason: .required), FieldIssue(field: "currency", reason: .invalid),
      ])
    ) {
      try await repository.createGroup(name: "", currency: "ZZZ", displayName: "Alice", key: WriteKey())
    }
  }

  @Test func aGroupICannotSeeIsNotFound() async throws {
    let id = try #require(UUID(uuidString: "0190b6c4-0000-7000-8000-00000000ffff"))
    let transport = try PathTransport([
      "GET /v1/groups/\(id.uuidString.lowercased())": .fixture("group-404-not-found", status: .notFound, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.problem(.notFound)) {
      try await repository.group(id: id)
    }
  }

  @Test func renameSendsTheVersionAndMapsTheResult() async throws {
    let id = try #require(UUID(uuidString: "01a0faaf-d1d6-7b92-93fe-46b51e96ba43"))
    let transport = try PathTransport([
      "PATCH /v1/groups/01a0faaf-d1d6-7b92-93fe-46b51e96ba43": .fixture("group-rename-200")
    ])
    let repository = try makeRepository(transport)

    let group = try await repository.renameGroup(id: id, name: "Goa 2026", version: 1, key: WriteKey())

    #expect(group.name == "Goa 2026")
    #expect(group.version == 2)
    let body = try JSONSerialization.jsonObject(with: try #require(await transport.sent.first).body) as? [String: Any]
    #expect(body?["version"] as? Int == 1)
  }

  @Test func aStaleVersionIsVersionConflict() async throws {
    let id = try #require(UUID(uuidString: "01a0faaf-d1d6-7b92-93fe-46b51e96ba43"))
    let transport = try PathTransport([
      "PATCH /v1/groups/01a0faaf-d1d6-7b92-93fe-46b51e96ba43": .fixture(
        "group-rename-409-version-conflict", status: .conflict, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.problem(.versionConflict)) {
      try await repository.renameGroup(id: id, name: "Goa x", version: 1, key: WriteKey())
    }
  }

  // MARK: Members (FR-M1, FR-M2, FR-G3)

  private static let groupPath = "/v1/groups/01a0fabc-fbe9-7151-9503-10d608c15583"
  private static let groupID = UUID(uuidString: "01a0fabc-fbe9-7151-9503-10d608c15583")!
  private static let bobID = UUID(uuidString: "01a0fabc-fcc1-7d77-92f4-db87ce484cc0")!

  @Test func addMemberSendsTheEmailAndMapsTheMember() async throws {
    let transport = try PathTransport(["POST \(Self.groupPath)/members": .fixture("member-add-201", status: .created)])
    let repository = try makeRepository(transport)

    let member = try await repository.addMember(
      groupID: Self.groupID, displayName: "Bob", email: "bob@example.com", key: WriteKey())

    #expect(member.displayName == "Bob")
    #expect(!member.isPlaceholder)
    #expect(member.joinSeq == 2)
    #expect(member.canBecomeAdmin)
    let body =
      try JSONSerialization.jsonObject(with: try #require(await transport.sent.first).body) as? [String: String]
    #expect(body == ["display_name": "Bob", "email": "bob@example.com"])
  }

  @Test func aNameOnlyPlaceholderSendsNoEmail() async throws {
    let transport = try PathTransport([
      "POST \(Self.groupPath)/members": .fixture("member-add-201-placeholder", status: .created)
    ])
    let repository = try makeRepository(transport)

    let member = try await repository.addMember(
      groupID: Self.groupID, displayName: "Grandma", email: nil, key: WriteKey())

    #expect(member.isPlaceholder)
    #expect(!member.canBecomeAdmin)
    let body =
      try JSONSerialization.jsonObject(with: try #require(await transport.sent.first).body) as? [String: String]
    #expect(body == ["display_name": "Grandma"])
  }

  @Test func aTakenDisplayNameIsAFieldIssue() async throws {
    let transport = try PathTransport([
      "POST \(Self.groupPath)/members": .fixture("member-add-400-taken", status: .badRequest, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.invalidFields([FieldIssue(field: "display_name", reason: .taken)])) {
      try await repository.addMember(groupID: Self.groupID, displayName: "bob", email: nil, key: WriteKey())
    }
  }

  @Test func makeAdminSendsTheVersion() async throws {
    let transport = try PathTransport([
      "PATCH \(Self.groupPath)/members/\(Self.bobID.uuidString.lowercased())": .fixture("member-admin-200")
    ])
    let repository = try makeRepository(transport)

    let member = try await repository.makeAdmin(
      groupID: Self.groupID, memberID: Self.bobID, version: 1, key: WriteKey())

    #expect(member.role == .admin)
    #expect(member.version == 2)
    let body = try JSONSerialization.jsonObject(with: try #require(await transport.sent.first).body) as? [String: Any]
    #expect(body?["role"] as? String == "admin")
    #expect(body?["version"] as? Int == 1)
  }

  @Test func aPlaceholderIsNotEligible() async throws {
    let transport = try PathTransport([
      "PATCH \(Self.groupPath)/members/\(Self.bobID.uuidString.lowercased())": .fixture(
        "member-admin-409-not-eligible", status: .conflict, problem: true)
    ])
    let repository = try makeRepository(transport)

    await #expect(throws: ServiceError.problem(.memberNotEligible)) {
      try await repository.makeAdmin(groupID: Self.groupID, memberID: Self.bobID, version: 1, key: WriteKey())
    }
  }

  private func makeRepository(_ transport: PathTransport) throws -> GroupsAPIRepository {
    let api = APISession(
      serverURL: try #require(URL(string: "http://localhost:8080")), transport: transport,
      tokens: InMemoryTokenStore(.sample))
    return GroupsAPIRepository(api: api)
  }
}

struct UserDefaultsPreferencesTests {
  @Test func defaultCurrencyFallsBackToTheLocaleThenRemembersTheChoice() throws {
    let suite = "dev.splits.tests.\(UUID().uuidString)"
    let defaults = try #require(UserDefaults(suiteName: suite))
    defer { defaults.removePersistentDomain(forName: suite) }
    let preferences = UserDefaultsPreferences(defaults: defaults, locale: Locale(identifier: "en_IN"))

    #expect(preferences.defaultCurrency() == "INR")
    preferences.setDefaultCurrency("EUR")
    #expect(preferences.defaultCurrency() == "EUR")
  }

  @Test func aStoredCodeTheServerRefusesFallsBackToTheRegion() throws {
    let suite = "dev.splits.tests.\(UUID().uuidString)"
    let defaults = try #require(UserDefaults(suiteName: suite))
    defer { defaults.removePersistentDomain(forName: suite) }
    defaults.set("XAU", forKey: "defaultCurrency")

    #expect(UserDefaultsPreferences(defaults: defaults, locale: Locale(identifier: "ja_JP")).defaultCurrency() == "JPY")
  }

  @Test func theLastReportGroupIsRemembered() throws {
    let suite = "dev.splits.tests.\(UUID().uuidString)"
    let defaults = try #require(UserDefaults(suiteName: suite))
    defer { defaults.removePersistentDomain(forName: suite) }
    let preferences = UserDefaultsPreferences(defaults: defaults)
    let id = UUID()

    #expect(preferences.lastReportGroupID() == nil)
    preferences.setLastReportGroupID(id)
    #expect(preferences.lastReportGroupID() == id)
  }

  @Test func aLocaleWithoutACurrencyFallsBackToUSD() throws {
    let suite = "dev.splits.tests.\(UUID().uuidString)"
    let defaults = try #require(UserDefaults(suiteName: suite))
    defer { defaults.removePersistentDomain(forName: suite) }

    #expect(UserDefaultsPreferences(defaults: defaults, locale: Locale(identifier: "en")).defaultCurrency() == "USD")
  }
}

/// Answers by "METHOD path?query" and records what was sent.
final class PathTransport: ClientTransport, Sendable {
  private let answers: [String: RecordingTransport.Behaviour]
  private let recorder = Recorder()

  init(_ answers: [String: RecordingTransport.Behaviour]) {
    self.answers = answers
  }

  var sent: [RecordingTransport.Sent] { get async { await recorder.sent } }

  func send(
    _ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String
  ) async throws -> (HTTPResponse, HTTPBody?) {
    let data = try await Data(collecting: body ?? HTTPBody(), upTo: 1 << 20)
    await recorder.record(RecordingTransport.Sent(request: request, body: data))
    let path = request.path ?? ""
    let key = "\(request.method.rawValue) \(path)"
    guard let answer = answers[key] else {
      return (HTTPResponse(status: .notImplemented), nil)
    }
    return try await RecordingTransport(behaviour: answer).send(
      request, body: nil, baseURL: baseURL, operationID: operationID)
  }

  private actor Recorder {
    var sent: [RecordingTransport.Sent] = []
    func record(_ s: RecordingTransport.Sent) { sent.append(s) }
  }
}
