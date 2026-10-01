import APIClient
import Domain
import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing

@testable import Data

/// Data is tested through the generated client with a fake transport: the
/// APIClient seam (doc 09). Responses are what the server sends per
/// api/openapi.yaml.
struct ServerHealthAPIRepositoryTests {
  @Test func succeedsWhenHealthzAnswersOK() async throws {
    let repository = try makeRepository(.answer(status: .ok, json: #"{"status":"ok"}"#))

    try await repository.checkHealth()
  }

  @Test func failsWithTheStatusWhenHealthzAnswersSomethingUndocumented() async throws {
    let repository = try makeRepository(.answer(status: .serviceUnavailable, json: "{}"))

    await #expect(throws: ServerHealthError.unexpectedStatus(503)) {
      try await repository.checkHealth()
    }
  }

  @Test func failsWithTheStatusWhenHealthzAnswersAnInternalError() async throws {
    let repository = try makeRepository(
      .answer(
        status: .internalServerError,
        json: #"{"type":"https://splits.dev/problems/internal","title":"Internal error","status":500}"#,
        contentType: "application/problem+json"))

    await #expect(throws: ServerHealthError.unexpectedStatus(500)) {
      try await repository.checkHealth()
    }
  }

  @Test func failsWhenTheServerCannotBeReached() async throws {
    let repository = try makeRepository(.fail(URLError(.cannotConnectToHost)))

    await #expect(throws: (any Error).self) {
      try await repository.checkHealth()
    }
  }

  private func makeRepository(_ behaviour: FakeTransport.Behaviour) throws -> ServerHealthAPIRepository {
    let client = Client(
      serverURL: try #require(URL(string: "http://localhost:8080")),
      transport: FakeTransport(behaviour: behaviour))
    return ServerHealthAPIRepository(client: client)
  }
}

private struct FakeTransport: ClientTransport {
  enum Behaviour: Sendable {
    case answer(status: HTTPResponse.Status, json: String, contentType: String = "application/json")
    case fail(any Error)
  }

  let behaviour: Behaviour

  func send(
    _ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String
  ) async throws -> (
    HTTPResponse, HTTPBody?
  ) {
    switch behaviour {
    case .answer(let status, let json, let contentType):
      var fields = HTTPFields()
      fields[.contentType] = contentType
      return (HTTPResponse(status: status, headerFields: fields), HTTPBody(json))
    case .fail(let error):
      throw error
    }
  }
}
