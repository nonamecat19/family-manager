import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Emits `true` when online (wifi, mobile, or ethernet), `false` when offline.
///
/// Uses [connectivity_plus] to detect network connection changes.
/// Note: this reports connection type, not actual internet reachability.
/// Always wrap Dio calls in try/catch regardless.
final connectivityProvider = StreamProvider<bool>((ref) {
  return Connectivity().onConnectivityChanged.map((results) {
    return results.any(
      (r) =>
          r == ConnectivityResult.wifi ||
          r == ConnectivityResult.mobile ||
          r == ConnectivityResult.ethernet,
    );
  });
});
