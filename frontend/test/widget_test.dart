import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:krasis/app.dart';

void main() {
  testWidgets('KrasisApp builds', (WidgetTester tester) async {
    await tester.pumpWidget(
      const ProviderScope(child: KrasisApp()),
    );
    await tester.pump();
    expect(find.byType(MaterialApp), findsOneWidget);
  });
}
