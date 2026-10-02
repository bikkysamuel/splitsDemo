import Foundation
import SwiftUI

/// Picks an ISO 4217 currency, shown as "INR · Indian Rupee" in the User's
/// language. The server accepts active codes only (Q91).
struct CurrencyPicker: View {
  let titleKey: String
  @Binding var selection: String

  var body: some View {
    Picker(selection: $selection) {
      ForEach(Self.codes(including: selection), id: \.self) { code in
        Text(verbatim: Self.label(code)).tag(code)
      }
    } label: {
      Text(LocalizedStringKey(titleKey), bundle: .module)
    }
    .navigationLinkPickerStyle()
  }

  /// Common codes sorted by name, plus the current one if it's unusual.
  static func codes(including current: String) -> [String] {
    var codes = Set(Locale.commonISOCurrencyCodes)
    codes.insert(current)
    return codes.sorted { label($0) < label($1) }
  }

  static func label(_ code: String) -> String {
    let name = Locale.current.localizedString(forCurrencyCode: code)
    return name.map { "\(code) · \($0)" } ?? code
  }
}

extension View {
  /// A picker that opens its own list (iOS); the default style on the Mac,
  /// where packages are also built for tests.
  fileprivate func navigationLinkPickerStyle() -> some View {
    #if os(iOS)
      self.pickerStyle(.navigationLink)
    #else
      self
    #endif
  }
}
