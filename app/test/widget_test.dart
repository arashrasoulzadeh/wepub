import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:app/main.dart';

void main() {
  testWidgets('smoke', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: WepubApp()));
    expect(find.text('wepub'), findsOneWidget);
  });
}
