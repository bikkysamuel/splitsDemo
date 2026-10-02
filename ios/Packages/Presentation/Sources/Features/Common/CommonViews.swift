import SwiftUI

/// String Catalog keys shared by many screens.
enum CommonKeys {
  static let cancel = "Cancel"
  static let ok = "OK"
  static let save = "Save"
  static let tryAgain = "Try again"
  static let loading = "Loading…"
  static let all = [cancel, ok, save, tryAgain, loading]
}

/// A screen's loading indicator, labelled for VoiceOver.
struct LoadingView: View {
  var body: some View {
    ProgressView()
      .accessibilityLabel(Text(LocalizedStringKey(CommonKeys.loading), bundle: .module))
  }
}

/// A screen that failed to load: the message and a retry.
struct FailedView: View {
  let messageKey: String
  let retry: () async -> Void

  var body: some View {
    ContentUnavailableView {
      Label {
        Text(LocalizedStringKey(messageKey), bundle: .module)
      } icon: {
        Image(systemName: "exclamationmark.triangle").accessibilityHidden(true)
      }
    } actions: {
      Button {
        Task { await retry() }
      } label: {
        Text(LocalizedStringKey(CommonKeys.tryAgain), bundle: .module).frame(minHeight: 44)
      }
      .buttonStyle(.bordered)
    }
  }
}
