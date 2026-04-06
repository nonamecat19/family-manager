import 'package:finance_tracker/core/network/connectivity_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('connectivityProvider', () {
    test('is a StreamProvider<bool>', () {
      // Verify the provider type compiles and is the expected type.
      // Mocking connectivity_plus streams requires platform channels,
      // so we verify the provider shape here.
      expect(connectivityProvider, isA<StreamProvider<bool>>());
    });
  });
}
