import 'dart:math';

class Ulid {
  static const _chars = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
  static String generate() {
    final rnd = Random.secure();
    final ts = DateTime.now().millisecondsSinceEpoch;
    final buf = StringBuffer();
    // timestamp base32 (10 chars)
    var t = ts;
    final tsChars = List.filled(10, '0');
    for (int i = 9; i >= 0; i--) {
      tsChars[i] = _chars[t % 32];
      t ~/= 32;
    }
    buf.writeAll(tsChars);
    for (int i = 0; i < 16; i++) {
      buf.write(_chars[rnd.nextInt(32)]);
    }
    return buf.toString();
  }
}
