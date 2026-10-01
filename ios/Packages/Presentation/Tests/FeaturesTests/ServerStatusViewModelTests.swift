import Domain
import Testing

@testable import Features

/// The iOS seam (ADR-0015): ViewModels are tested through their public state
/// with a fake repository standing in for Data.
@MainActor
struct ServerStatusViewModelTests {
  @Test func startsUnknownBeforeAnyCheck() {
    let viewModel = ServerStatusViewModel(repository: FakeServerHealthRepository(outcome: .success(())))

    #expect(viewModel.status == .unknown)
  }

  @Test func reportsReachableWhenTheServerAnswers() async {
    let viewModel = ServerStatusViewModel(repository: FakeServerHealthRepository(outcome: .success(())))

    await viewModel.refresh()

    #expect(viewModel.status == .reachable)
  }

  @Test func reportsUnreachableWhenTheCheckFails() async {
    let viewModel = ServerStatusViewModel(
      repository: FakeServerHealthRepository(outcome: .failure(FakeError.offline)))

    await viewModel.refresh()

    #expect(viewModel.status == .unreachable)
  }

  @Test func recoversToReachableOnTheNextSuccessfulCheck() async {
    let repository = FakeServerHealthRepository(outcome: .failure(FakeError.offline))
    let viewModel = ServerStatusViewModel(repository: repository)
    await viewModel.refresh()

    await repository.setOutcome(.success(()))
    await viewModel.refresh()

    #expect(viewModel.status == .reachable)
  }
}

private enum FakeError: Error {
  case offline
}

private actor FakeServerHealthRepository: ServerHealthRepository {
  private var outcome: Result<Void, any Error>

  init(outcome: Result<Void, any Error>) {
    self.outcome = outcome
  }

  func setOutcome(_ outcome: Result<Void, any Error>) {
    self.outcome = outcome
  }

  func checkHealth() async throws {
    try outcome.get()
  }
}
