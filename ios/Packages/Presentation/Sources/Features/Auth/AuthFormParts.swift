import SwiftUI

/// A form field's error, read by VoiceOver right after the field.
struct FieldErrorText: View {
  let key: String?

  var body: some View {
    if let key {
      Text(LocalizedStringKey(key), bundle: .module)
        .font(.footnote)
        .foregroundStyle(.red)
    }
  }
}

/// The screen-level message above the submit button.
struct FormMessage: View {
  let key: String?
  var isError = true

  var body: some View {
    if let key {
      Label {
        Text(LocalizedStringKey(key), bundle: .module)
      } icon: {
        Image(systemName: isError ? "exclamationmark.triangle.fill" : "checkmark.circle.fill")
          .foregroundStyle(isError ? .red : .green)
          .accessibilityHidden(true)
      }
      .font(.callout)
    }
  }
}

/// The primary button of an auth form: full width, at least 44 pt tall,
/// with a progress indicator while submitting.
struct SubmitButton: View {
  let titleKey: String
  let isSubmitting: Bool
  let isEnabled: Bool
  let action: () async -> Void

  var body: some View {
    Button {
      Task { await action() }
    } label: {
      HStack {
        Spacer()
        if isSubmitting {
          // VoiceOver keeps reading the button's title while it works.
          ProgressView()
            .accessibilityLabel(Text(LocalizedStringKey(titleKey), bundle: .module))
        } else {
          Text(LocalizedStringKey(titleKey), bundle: .module)
        }
        Spacer()
      }
      .frame(minHeight: 44)
    }
    .buttonStyle(.borderedProminent)
    .disabled(!isEnabled)
    .listRowInsets(EdgeInsets())
    .listRowBackground(Color.clear)
  }
}
