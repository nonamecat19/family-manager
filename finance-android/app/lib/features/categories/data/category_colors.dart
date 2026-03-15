import 'dart:ui';

/// Curated category color palette -- 16 colors with good contrast and harmony.
///
/// Used by the color picker bottom sheet and for display throughout the app.
const List<Color> categoryColors = [
  Color(0xFFEF5350), // Red
  Color(0xFFEC407A), // Pink
  Color(0xFFAB47BC), // Purple
  Color(0xFF7E57C2), // Deep Purple
  Color(0xFF5C6BC0), // Indigo
  Color(0xFF42A5F5), // Blue
  Color(0xFF29B6F6), // Light Blue
  Color(0xFF26C6DA), // Cyan
  Color(0xFF26A69A), // Teal
  Color(0xFF66BB6A), // Green
  Color(0xFF9CCC65), // Light Green
  Color(0xFFD4E157), // Lime
  Color(0xFFFFCA28), // Amber
  Color(0xFFFFA726), // Orange
  Color(0xFFFF7043), // Deep Orange
  Color(0xFF8D6E63), // Brown
];

/// Parses a hex color string like "#FF5722" to a [Color].
Color parseHexColor(String hex) {
  final hexValue = hex.replaceFirst('#', '');
  return Color(int.parse('FF$hexValue', radix: 16));
}

/// Converts a [Color] to a hex string like "#FF5722".
String colorToHex(Color color) {
  // ignore: deprecated_member_use
  final value = color.value;
  return '#${(value & 0xFFFFFF).toRadixString(16).padLeft(6, '0').toUpperCase()}';
}
