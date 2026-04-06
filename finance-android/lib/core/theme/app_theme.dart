import 'package:finance_tracker/core/theme/app_colors.dart';
import 'package:flutter/material.dart';

/// Builds the app [ThemeData] for the given [brightness].
///
/// Uses Material 3 with zero elevation throughout for flat design.
ThemeData buildAppTheme(Brightness brightness) {
  final colorScheme = ColorScheme.fromSeed(
    seedColor: AppColors.seed,
    brightness: brightness,
  );

  return ThemeData(
    useMaterial3: true,
    colorScheme: colorScheme,
    appBarTheme: const AppBarTheme(
      elevation: 0,
      scrolledUnderElevation: 0,
    ),
    cardTheme: const CardThemeData(
      elevation: 0,
    ),
    navigationBarTheme: const NavigationBarThemeData(
      elevation: 0,
      shadowColor: Colors.transparent,
    ),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      elevation: 0,
      highlightElevation: 0,
      hoverElevation: 0,
      focusElevation: 0,
    ),
    dialogTheme: const DialogThemeData(
      elevation: 0,
    ),
    bottomSheetTheme: const BottomSheetThemeData(
      elevation: 0,
    ),
  );
}
