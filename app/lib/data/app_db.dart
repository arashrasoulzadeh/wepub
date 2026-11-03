import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'dart:io';

part 'app_db.g.dart';

class Ops extends Table {
  IntColumn get seq => integer().autoIncrement()();
  TextColumn get id => text()();
  TextColumn get deviceId => text().nullable()();
  TextColumn get hlc => text().nullable()();
  TextColumn get entity => text()();
  TextColumn get entityId => text()();
  TextColumn get field => text()();
  BlobColumn get value => blob().nullable()();
}

@DriftDatabase(tables: [Ops])
class AppDatabase extends _$AppDatabase {
  AppDatabase() : super(_openConnection());
  @override
  int get schemaVersion => 1;
}

QueryExecutor _openConnection() {
  return LazyDatabase(() async {
    final dir = await getApplicationDocumentsDirectory();
    final file = File(p.join(dir.path, 'wepub.db'));
    return NativeDatabase(file);
  });
}
