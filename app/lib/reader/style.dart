class ReaderStyle {
  final String theme;
  final String fontFamily;
  final double fontSize;
  final double lineHeight;
  final double margins;
  final bool justify;
  final bool publisherStyles;
  ReaderStyle({
    this.theme = 'light',
    this.fontFamily = 'serif',
    this.fontSize = 18,
    this.lineHeight = 1.5,
    this.margins = 16,
    this.justify = false,
    this.publisherStyles = true,
  });
  String toCss() {
    return '--theme:$theme;--font:$fontFamily;--size:${fontSize}px;--lh:$lineHeight;--margins:${margins}px;';
  }
}
