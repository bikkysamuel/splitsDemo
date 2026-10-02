import Domain
import Foundation
import SwiftUI

/// Picks one of the active ISO 4217 currencies the server accepts (Q91),
/// shown as "INR · Indian Rupee" in the User's language.
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

  /// The active codes sorted by name, plus the current one if it's
  /// unusual.
  static func codes(including current: String) -> [String] {
    var codes = Set(Currency.activeCodes)
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
