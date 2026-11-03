class OrSet<T> {
  final Map<T, Set<String>> _adds = {};
  final Set<String> _tombstones = {};
  void add(T v, String tag) {
    _adds.putIfAbsent(v, () => {}).add(tag);
  }
  void remove(T v, String tag) {
    final tags = _adds[v];
    if (tags != null) {
      for (final t in List.of(tags)) {
        _tombstones.add(t);
      }
      tags.clear();
    }
  }
  Set<T> elements() {
    final out = <T>{};
    _adds.forEach((v, tags) {
      if (tags.any((t) => !_tombstones.contains(t))) {
        out.add(v);
      }
    });
    return out;
  }
}
