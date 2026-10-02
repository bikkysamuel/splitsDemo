import Domain
import SwiftUI

/// Settings (FR-U5). For now: the signed-in account and Sign out; theme,
/// currency and About come with later tickets.
struct SettingsView: View {
  let user: User
  let session: AppSession
  @Bindable var viewModel: SettingsViewModel
  @State private var confirmsSignOut = false
  @State private var isSigningOut = false

  var body: some View {
    List {
      Section {
        CurrencyPicker(titleKey: Self.defaultCurrency, selection: $viewModel.defaultCurrency)
      } header: {
        Text(LocalizedStringKey(Self.newGroups), bundle: .module)
      } footer: {
        Text(LocalizedStringKey(Self.defaultCurrencyFooter), bundle: .module)
      }
      Section {
        LabeledContent {
          Text(verbatim: user.email)
        } label: {
          Text(LocalizedStringKey(Self.signedInAs), bundle: .module)
        }
        Button(role: .destructive) {
          confirmsSignOut = true
        } label: {
          HStack {
            Text(LocalizedStringKey(Self.signOut), bundle: .module)
            if isSigningOut {
              Spacer()
              ProgressView()
                .accessibilityLabel(Text(LocalizedStringKey(Self.signingOut), bundle: .module))
            }
          }
          .frame(minHeight: 44)
        }
        .disabled(isSigningOut)
      } header: {
        Text(LocalizedStringKey(Self.account), bundle: .module)
      }
    }
    .navigationTitle(Text(LocalizedStringKey(MainTabView.settings), bundle: .module))
    .confirmationDialog(
      Text(LocalizedStringKey(Self.confirmTitle), bundle: .module), isPresented: $confirmsSignOut,
      titleVisibility: .visible
    ) {
      Button(role: .destructive) {
        isSigningOut = true
        Task {
          await session.signOut()
          isSigningOut = false
        }
      } label: {
        Text(LocalizedStringKey(Self.signOut), bundle: .module)
      }
    }
  }

  nonisolated static let account = "Account"
  nonisolated static let signedInAs = "Signed in as"
  nonisolated static let signOut = "Sign out"
  nonisolated static let confirmTitle = "Sign out of Splits on this iPhone?"
  nonisolated static let signingOut = "Signing out…"
  nonisolated static let newGroups = "New Groups"
  nonisolated static let defaultCurrency = "Default currency"
  nonisolated static let defaultCurrencyFooter = "Pre-fills the Group Currency when you create a Group."
  nonisolated static let allKeys = [
    account, signedInAs, signOut, confirmTitle, signingOut, newGroups, defaultCurrency, defaultCurrencyFooter,
  ]
}
