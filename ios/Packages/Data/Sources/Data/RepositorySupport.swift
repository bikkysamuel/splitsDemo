import APIClient
import Domain
import Foundation
import OpenAPIRuntime

// Helpers every API repository uses to turn client outcomes into
// `ServiceError`s.

/// Runs a client call, mapping transport and decoding failures.
func send<Output>(_ call: () async throws -> Output) async throws(ServiceError) -> Output {
  do {
    return try await call()
  } catch {
    throw ServiceErrorMapper.transportError(error)
  }
}

/// Reads a response body; a body that doesn't match the contract is a
/// server bug.
func decoding<Body>(_ read: () throws -> Body) throws(ServiceError) -> Body {
  do {
    return try read()
  } catch {
    throw .unexpected(status: nil)
  }
}

/// The `ServiceError` of a documented problem+json response.
func problem(_ status: Int, _ read: () throws -> Components.Schemas.Problem) -> ServiceError {
  guard let p = try? read() else { return .unexpected(status: status) }
  return ServiceErrorMapper.error(problem: p, status: status)
}

/// A UUID the contract promises; anything else is a server bug.
func uuid(_ text: String) throws(ServiceError) -> UUID {
  guard let id = UUID(uuidString: text) else { throw .unexpected(status: nil) }
  return id
}
