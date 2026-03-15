import 'package:flutter/material.dart';

/// Maps icon name strings (stored in DB) to Flutter [IconData].
///
/// This is the single source of truth for available category icons.
/// The icon picker displays these in a grid; category display widgets
/// resolve icon names through this map.
const Map<String, IconData> categoryIcons = {
  // Food & Drink
  'restaurant': Icons.restaurant,
  'local_cafe': Icons.local_cafe,
  'local_bar': Icons.local_bar,
  'local_grocery_store': Icons.local_grocery_store,
  // Transport
  'directions_car': Icons.directions_car,
  'directions_bus': Icons.directions_bus,
  'local_gas_station': Icons.local_gas_station,
  'flight': Icons.flight,
  // Shopping
  'shopping_cart': Icons.shopping_cart,
  'shopping_bag': Icons.shopping_bag,
  'storefront': Icons.storefront,
  // Home
  'home': Icons.home,
  'electrical_services': Icons.electrical_services,
  'plumbing': Icons.plumbing,
  // Health
  'local_hospital': Icons.local_hospital,
  'fitness_center': Icons.fitness_center,
  'medication': Icons.medication,
  // Entertainment
  'movie': Icons.movie,
  'sports_esports': Icons.sports_esports,
  'music_note': Icons.music_note,
  // Education
  'school': Icons.school,
  'menu_book': Icons.menu_book,
  // Personal
  'checkroom': Icons.checkroom,
  'content_cut': Icons.content_cut,
  // Finance
  'savings': Icons.savings,
  'account_balance': Icons.account_balance,
  'credit_card': Icons.credit_card,
  // Misc
  'pets': Icons.pets,
  'child_care': Icons.child_care,
  'card_giftcard': Icons.card_giftcard,
  'phone_android': Icons.phone_android,
  'wifi': Icons.wifi,
  'more_horiz': Icons.more_horiz,
};
