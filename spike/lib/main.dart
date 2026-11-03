import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';

const _port = int.fromEnvironment('PORT', defaultValue: 8765);
final _server = InAppLocalhostServer(port: _port, documentRoot: 'assets');

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await _server.start();
  runApp(const SpikeApp());
}

class SpikeApp extends StatelessWidget {
  const SpikeApp({super.key});

  @override
  Widget build(BuildContext context) {
    const auto = bool.fromEnvironment('AUTO');
    const page = String.fromEnvironment('PAGE', defaultValue: 'reader');
    final url = 'http://localhost:$_port/reader/$page.html${auto ? '?auto=1' : ''}';
    return MaterialApp(
      home: Scaffold(
        body: SafeArea(
          child: InAppWebView(
            initialUrlRequest: URLRequest(url: WebUri(url)),
            initialSettings: InAppWebViewSettings(isInspectable: true),
            onConsoleMessage: (_, m) => debugPrint('JS ${m.message}'),
          ),
        ),
      ),
    );
  }
}
