import Domain
import Foundation
import Testing

@testable import Features

@MainActor
struct HomeViewModelTests {
  @Test func loadsTheGroups() async {
    let repository = FakeGroupsRepository()
    await repository.set(groups: .success([Group.trip.summary]))
    let viewModel = HomeViewModel(repository: repository)
    #expect(viewModel.state == .loading)

    await viewModel.load()

    #expect(viewModel.state == .loaded([Group.trip.summary]))
  }

  @Test func showsAnErrorWhenTheFirstLoadFails() async {
    let repository = FakeGroupsRepository()
    await repository.set(groups: .failure(.unreachable))
    let viewModel = HomeViewModel(repository: repository)

    await viewModel.load()

    #expect(viewModel.state == .failed(ServiceErrorMessage.unreachable))
  }

  @Test func aFailedRefreshKeepsTheList() async {
    let repository = FakeGroupsRepository()
    await repository.set(groups: .success([Group.trip.summary]))
    let viewModel = HomeViewModel(repository: repository)
    await viewModel.load()

    await repository.set(groups: .failure(.unreachable))
    await viewModel.load()

    #expect(viewModel.state == .loaded([Group.trip.summary]))
  }

  @Test func aCreatedGroupJoinsTheList() async {
    let viewModel = HomeViewModel(repository: FakeGroupsRepository())
    await viewModel.load()

    viewModel.didCreate(.trip)

    #expect(viewModel.state == .loaded([Group.trip.summary]))
  }
}

@MainActor
struct CreateGroupViewModelTests {
  @Test func theCurrencyIsPrefilledFromSettings() {
    let viewModel = CreateGroupViewModel(
      repository: FakeGroupsRepository(), preferences: FakePreferences(currency: "EUR"))

    #expect(viewModel.currency == "EUR")
  }

  @Test func createsTheGroup() async {
    let repository = FakeGroupsRepository()
    let viewModel = CreateGroupViewModel(repository: repository, preferences: FakePreferences())
    viewModel.name = "Goa trip"
    viewModel.displayName = "Alice"

    let group = await viewModel.submit()

    #expect(group == .trip)
    let created = await repository.created
    #expect(created.count == 1 && created[0] == ("Goa trip", "INR", "Alice"))
  }

  @Test func needsANameAndADisplayName() async {
    let repository = FakeGroupsRepository()
    let viewModel = CreateGroupViewModel(repository: repository, preferences: FakePreferences())
    viewModel.name = "  "
    viewModel.displayName = "Alice"

    #expect(!viewModel.canSubmit)
    #expect(await viewModel.submit() == nil)
    #expect(await repository.created.isEmpty)
  }

  @Test func showsRefusedFieldsUnderEachField() async {
    let repository = FakeGroupsRepository()
    await repository.set(create: .failure(.invalidFields([FieldIssue(field: "currency", reason: .invalid)])))
    let viewModel = CreateGroupViewModel(repository: repository, preferences: FakePreferences())
    viewModel.name = "Trip"
    viewModel.displayName = "Alice"

    #expect(await viewModel.submit() == nil)

    #expect(viewModel.errors["currency"] == "Choose a currency.")
  }

  // NFR-R1: retrying the same submission reuses its key; edited input and
  // a new submission after success get fresh ones.
  @Test func aRetryReusesTheIdempotencyKey() async {
    let repository = FakeGroupsRepository()
    await repository.set(create: .failure(.unreachable))
    let viewModel = CreateGroupViewModel(repository: repository, preferences: FakePreferences())
    viewModel.name = "Trip"
    viewModel.displayName = "Alice"

    _ = await viewModel.submit()
    _ = await viewModel.submit()
    viewModel.name = "Trip 2"
    _ = await viewModel.submit()

    let keys = await repository.keys
    #expect(keys.count == 3 && keys[0] == keys[1] && keys[2] != keys[1])
  }

  @Test func theGroupLimitIsShown() async {
    let repository = FakeGroupsRepository()
    await repository.set(create: .failure(.problem(.groupLimitReached)))
    let viewModel = CreateGroupViewModel(repository: repository, preferences: FakePreferences())
    viewModel.name = "Trip"
    viewModel.displayName = "Alice"

    _ = await viewModel.submit()

    #expect(viewModel.errors.message == "You're already in 200 Groups, the most allowed.")
  }
}

@MainActor
struct GroupViewModelTests {
  @Test func loadsTheGroup() async {
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: FakeGroupsRepository())

    await viewModel.load()

    #expect(viewModel.state == .loaded(.trip))
    #expect(viewModel.canRename)
  }

  @Test func aGroupICannotSeeShowsNotFound() async {
    let repository = FakeGroupsRepository()
    await repository.set(group: .failure(.problem(.notFound)))
    let viewModel = GroupViewModel(groupID: UUID(), repository: repository)

    await viewModel.load()

    #expect(viewModel.state == .failed(ServiceErrorMessage.key(for: .notFound)))
  }

  @Test func onlyAdminsMayRename() async {
    let repository = FakeGroupsRepository()
    await repository.set(group: .success(Group.trip.with(name: "Goa trip", version: 1, role: .member)))
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)

    await viewModel.load()

    #expect(!viewModel.canRename)
  }

  @Test func renamesWithTheVersionLastRead() async {
    let repository = FakeGroupsRepository()
    await repository.set(renames: [.success(Group.trip.with(name: "Goa 2026", version: 2))])
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)
    await viewModel.load()

    await viewModel.rename(to: "Goa 2026")

    let renames = await repository.renames
    #expect(renames.count == 1 && renames[0] == ("Goa 2026", 1))
    #expect(viewModel.state.value?.name == "Goa 2026")
    #expect(viewModel.changeError == nil)
  }

  // NFR-R4: a stale version shows the conflict and reloads the Group.
  @Test func aVersionConflictReloads() async {
    let repository = FakeGroupsRepository()
    await repository.set(renames: [.failure(.problem(.versionConflict))])
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)
    await viewModel.load()
    await repository.set(group: .success(Group.trip.with(name: "Renamed elsewhere", version: 2)))

    await viewModel.rename(to: "Mine")

    #expect(viewModel.changeError == "Someone else changed this just now. Reload and try again.")
    #expect(viewModel.state.value?.name == "Renamed elsewhere")
  }
}

@MainActor
struct SettingsViewModelTests {
  @Test func theDefaultCurrencyIsRememberedInPreferences() {
    let preferences = FakePreferences(currency: "INR")
    let viewModel = SettingsViewModel(preferences: preferences)
    #expect(viewModel.defaultCurrency == "INR")

    viewModel.defaultCurrency = "JPY"

    #expect(preferences.currency == "JPY")
  }
}

@MainActor
struct AddMemberViewModelTests {
  @Test func addsByEmailTrimmed() async {
    let repository = FakeGroupsRepository()
    let viewModel = AddMemberViewModel(groupID: Group.trip.id, repository: repository)
    viewModel.displayName = "Bob"
    viewModel.email = " bob@example.com "

    #expect(await viewModel.submit() == .bob)

    let added = await repository.added
    #expect(added.count == 1 && added[0] == ("Bob", "bob@example.com"))
  }

  @Test func anEmptyEmailAddsANameOnlyPlaceholder() async {
    let repository = FakeGroupsRepository()
    let viewModel = AddMemberViewModel(groupID: Group.trip.id, repository: repository)
    viewModel.displayName = "Grandma"
    viewModel.email = "  "

    _ = await viewModel.submit()

    let added = await repository.added
    #expect(added.count == 1 && added[0] == ("Grandma", nil))
  }

  @Test func takenFieldsShowUnderEachField() async {
    let repository = FakeGroupsRepository()
    await repository.set(
      add: .failure(
        .invalidFields([
          FieldIssue(field: "display_name", reason: .taken), FieldIssue(field: "email", reason: .taken),
        ])))
    let viewModel = AddMemberViewModel(groupID: Group.trip.id, repository: repository)
    viewModel.displayName = "Bob"
    viewModel.email = "bob@example.com"

    #expect(await viewModel.submit() == nil)

    #expect(viewModel.errors["display_name"] == "Someone in this Group already has this name.")
    #expect(viewModel.errors["email"] == "Someone in this Group already has this email.")
  }

  @Test func theMemberLimitIsShown() async {
    let repository = FakeGroupsRepository()
    await repository.set(add: .failure(.problem(.memberLimitReached)))
    let viewModel = AddMemberViewModel(groupID: Group.trip.id, repository: repository)
    viewModel.displayName = "One too many"

    _ = await viewModel.submit()

    #expect(viewModel.errors.message == "This Group already has 50 Members, the most allowed.")
  }
}

@MainActor
struct GrantAdminTests {
  @Test func anAdminMayMakeLinkedMembersAdmins() async {
    let repository = FakeGroupsRepository()
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)
    await viewModel.load()

    #expect(viewModel.canMakeAdmin(.bob))
    #expect(!viewModel.canMakeAdmin(.grandma))
  }

  @Test func aMemberWhoIsNotAnAdminMayNot() async {
    let repository = FakeGroupsRepository()
    await repository.set(group: .success(Group.trip.with(name: "Goa trip", version: 1, role: .member)))
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)
    await viewModel.load()

    #expect(!viewModel.canMakeAdmin(.bob))
  }

  @Test func makeAdminSendsTheMembersVersion() async {
    let repository = FakeGroupsRepository()
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)
    await viewModel.load()

    await viewModel.makeAdmin(.bob)

    let grants = await repository.adminGrants
    #expect(grants.count == 1 && grants[0] == (Member.bob.id, 3))
    #expect(viewModel.changeError == nil)
  }

  @Test func aRefusalIsShown() async {
    let repository = FakeGroupsRepository()
    await repository.set(admin: .failure(.problem(.memberNotEligible)))
    let viewModel = GroupViewModel(groupID: Group.trip.id, repository: repository)
    await viewModel.load()

    await viewModel.makeAdmin(.grandma)

    #expect(viewModel.changeError == "Only a Member who has an account can be an Admin.")
  }
}
