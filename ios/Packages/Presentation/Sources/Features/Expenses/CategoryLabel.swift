import Domain
import SwiftUI

/// A Category's localized name and symbol (D7).
struct CategoryLabel: View {
  let category: Domain.Category

  var body: some View {
    Label {
      Text(LocalizedStringKey(Self.nameKey(category)), bundle: .module)
    } icon: {
      Image(systemName: Self.symbol(category)).accessibilityHidden(true)
    }
  }

  nonisolated static func nameKey(_ c: Domain.Category) -> String {
    switch c {
    case .foodDrink: "Food & drink"
    case .groceries: "Groceries"
    case .transport: "Transport"
    case .accommodation: "Accommodation"
    case .rent: "Rent"
    case .utilities: "Utilities"
    case .entertainment: "Entertainment"
    case .shopping: "Shopping"
    case .health: "Health"
    case .travel: "Travel"
    case .other: "Other"
    }
  }

  nonisolated static func symbol(_ c: Domain.Category) -> String {
    switch c {
    case .foodDrink: "fork.knife"
    case .groceries: "cart"
    case .transport: "car"
    case .accommodation: "bed.double"
    case .rent: "house"
    case .utilities: "bolt"
    case .entertainment: "theatermasks"
    case .shopping: "bag"
    case .health: "cross.case"
    case .travel: "airplane"
    case .other: "square.grid.2x2"
    }
  }
}
