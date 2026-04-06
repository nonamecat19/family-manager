// @dart=3.6
// ignore_for_file: directives_ordering
// build_runner >=2.4.16
// ignore_for_file: no_leading_underscores_for_library_prefixes
import 'package:build_runner/src/build_plan/builder_factories.dart' as _i1;
import 'package:build_modules/builders.dart' as _i2;
import 'package:build_web_compilers/builders.dart' as _i3;
import 'dart:io' as _i4;
import 'package:build_runner/src/bootstrap/processes.dart' as _i5;

final _builderFactories = _i1.BuilderFactories(
  {
    'build_modules:module_library': [_i2.moduleLibraryBuilder],
    'build_web_compilers:dart2js_modules': [
      _i3.dart2jsMetaModuleBuilder,
      _i3.dart2jsMetaModuleCleanBuilder,
      _i3.dart2jsModuleBuilder,
    ],
    'build_web_compilers:dart2wasm_modules': [
      _i3.dart2wasmMetaModuleBuilder,
      _i3.dart2wasmMetaModuleCleanBuilder,
      _i3.dart2wasmModuleBuilder,
    ],
    'build_web_compilers:ddc': [
      _i3.ddcKernelBuilder,
      _i3.ddcBuilder,
    ],
    'build_web_compilers:ddc_modules': [
      _i3.ddcMetaModuleBuilder,
      _i3.ddcMetaModuleCleanBuilder,
      _i3.ddcModuleBuilder,
    ],
    'build_web_compilers:entrypoint': [_i3.webEntrypointBuilder],
    'build_web_compilers:entrypoint_marker': [_i3.webEntrypointMarkerBuilder],
    'build_web_compilers:sdk_js': [
      _i3.sdkJsCompile,
      _i3.sdkJsCopyRequirejs,
    ],
  },
  postProcessBuilderFactories: {
    'build_modules:module_cleanup': _i2.moduleCleanup,
    'build_web_compilers:dart2js_archive_extractor':
        _i3.dart2jsArchiveExtractor,
    'build_web_compilers:dart_source_cleanup': _i3.dartSourceCleanup,
  },
);
void main(List<String> args) async {
  _i4.exitCode = await _i5.ChildProcess.run(
    args,
    _builderFactories,
  )!;
}
