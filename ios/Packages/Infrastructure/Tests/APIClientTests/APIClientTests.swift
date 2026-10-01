import Foundation
import Testing

@testable import APIClient

struct URLSessionFactoryTests {
  // ADR-0005: no domain data on the device, so no HTTP disk cache either.
  @Test func sessionKeepsNoHTTPCache() {
    let session = URLSessionFactory.makeSession()

    #expect(session.configuration.urlCache == nil)
    #expect(session.configuration.requestCachePolicy == .reloadIgnoringLocalCacheData)
  }
}

struct APIConfigurationTests {
  @Test func readsTheBaseURLFromTheInfoDictionary() throws {
    let configuration = try APIConfiguration(
      infoDictionary: ["SplitsAPIBaseURL": "http://localhost:8080"], requiresHTTPS: false)

    #expect(configuration.baseURL == URL(string: "http://localhost:8080"))
  }

  // Q75: Release builds require HTTPS.
  @Test func rejectsPlainHTTPWhenHTTPSIsRequired() {
    #expect(throws: APIConfigurationError.insecureBaseURL("http://localhost:8080")) {
      try APIConfiguration(
        infoDictionary: ["SplitsAPIBaseURL": "http://localhost:8080"], requiresHTTPS: true)
    }
  }

  @Test func acceptsHTTPSWhenHTTPSIsRequired() throws {
    let configuration = try APIConfiguration(
      infoDictionary: ["SplitsAPIBaseURL": "https://api.example.com"], requiresHTTPS: true)

    #expect(configuration.baseURL == URL(string: "https://api.example.com"))
  }

  @Test func rejectsAMissingBaseURL() {
    #expect(throws: APIConfigurationError.missingBaseURL) {
      try APIConfiguration(infoDictionary: [:], requiresHTTPS: false)
    }
  }

  @Test(arguments: ["", "localhost:8080", "not a url"])
  func rejectsABaseURLWithoutSchemeAndHost(value: String) {
    #expect(throws: APIConfigurationError.invalidBaseURL(value)) {
      try APIConfiguration(infoDictionary: ["SplitsAPIBaseURL": value], requiresHTTPS: false)
    }
  }
}
