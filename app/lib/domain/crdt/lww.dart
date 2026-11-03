class LwwValue<T> {
  final T value;
  final String hlc;
  final String device;
  LwwValue(this.value, this.hlc, this.device);
}

class LwwRegister<T> {
  LwwValue<T>? _current;
  LwwValue<T>? get current => _current;
  bool apply(LwwValue<T> v) {
    if (_current == null || _current!.hlc.compareTo(v.hlc) < 0 ||
        (_current!.hlc == v.hlc && _current!.device.compareTo(v.device) < 0)) {
      _current = v;
      return true;
    }
    return false;
  }
}
