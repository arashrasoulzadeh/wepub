import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

void main() {
  runApp(const ProviderScope(child: WepubApp()));
}

class WepubApp extends StatelessWidget {
  const WepubApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'wepub',
      home: const Scaffold(body: Center(child: Text('wepub'))),
    );
  }
}
