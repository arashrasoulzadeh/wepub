class HLC implements Comparable<HLC> {
  final int wallMs;
  final int counter;
  final String device;
  HLC(this.wallMs, this.counter, this.device);
  @override
  int compareTo(HLC o) {
    if (wallMs != o.wallMs) return wallMs.compareTo(o.wallMs);
    if (counter != o.counter) return counter.compareTo(o.counter);
    return device.compareTo(o.device);
  }
  @override
  String toString() =>
      '${wallMs.toString().padLeft(13, '0')}-${counter.toRadixString(16).padLeft(4, '0')}-$device';
  static HLC parse(String s) {
    final parts = s.split('-');
    return HLC(int.parse(parts[0]), int.parse(parts[1], radix: 16), parts[2]);
  }
}
