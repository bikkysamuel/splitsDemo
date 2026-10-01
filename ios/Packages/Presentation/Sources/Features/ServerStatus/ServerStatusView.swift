import SwiftUI

/// One line saying whether the server is reachable, with a retry button.
public struct ServerStatusView: View {
  let viewModel: ServerStatusViewModel

  public init(viewModel: ServerStatusViewModel) {
    self.viewModel = viewModel
  }

  public var body: some View {
    let appearance = ServerStatusAppearance(viewModel.status)
    VStack(spacing: 12) {
      Label {
        Text(LocalizedStringKey(appearance.titleKey), bundle: .module)
      } icon: {
        Image(systemName: appearance.symbol)
          .foregroundStyle(appearance.tint)
          .accessibilityHidden(true)
      }
      .font(.body)
      .accessibilityElement(children: .combine)

      Button {
        Task { await viewModel.refresh() }
      } label: {
        Text(LocalizedStringKey(Self.retryKey), bundle: .module)
          .frame(minHeight: 44)  // NFR-A2: at least 44×44 pt
      }
      .buttonStyle(.bordered)
      .disabled(viewModel.status == .checking)
    }
    .task { await viewModel.refresh() }
  }

  /// String Catalog keys; the catalog holds each one's comment.
  static let retryKey = "Check again"
}

/// How each status looks: catalog key, SF Symbol and tint.
struct ServerStatusAppearance {
  let titleKey: String
  let symbol: String
  let tint: Color

  init(_ status: ServerStatusViewModel.Status) {
    switch status {
    case .unknown, .checking:
      titleKey = "Checking the server…"
      symbol = "ellipsis.circle"
      tint = .secondary
    case .reachable:
      titleKey = "Server reachable"
      symbol = "checkmark.circle.fill"
      tint = .green
    case .unreachable:
      titleKey = "Server unreachable"
      symbol = "xmark.octagon.fill"
      tint = .red
    }
  }
}
