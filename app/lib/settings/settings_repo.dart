class SettingsRepo {
  final Map<String, Object?> _store = {};
  T? get<T>(String key) => _store[key] as T?;
  void set<T>(String key, T value) { _store[key] = value; }
}
