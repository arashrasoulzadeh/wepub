import 'dart:convert';
import 'package:archive/archive.dart';
import 'package:xml/xml.dart';

class EpubMeta {
  final String? title;
  final String? author;
  final String? language;
  EpubMeta({this.title, this.author, this.language});
}

class EpubParser {
  static EpubMeta parse(List<int> bytes) {
    final archive = ZipDecoder().decodeBytes(bytes);
    // find container.xml
    ArchiveFile? container;
    for (final f in archive) {
      if (f.name == 'META-INF/container.xml') container = f;
    }
    if (container == null) return EpubMeta();
    final doc = XmlDocument.parse(utf8.decode(container.content as List<int>));
    final rootfile = doc.findAllElements('rootfile').first;
    final opfPath = rootfile.getAttribute('full-path')!;
    ArchiveFile? opf;
    for (final f in archive) {
      if (f.name == opfPath) opf = f;
    }
    if (opf == null) return EpubMeta();
    final opfDoc = XmlDocument.parse(utf8.decode(opf.content as List<int>));
    String? title;
    String? author;
    String? lang;
    for (final e in opfDoc.findAllElements('title')) { title = e.innerText; break; }
    for (final e in opfDoc.findAllElements('creator')) { author = e.innerText; break; }
    final pkg = opfDoc.findAllElements('package').first;
    lang = pkg.getAttribute('xml:lang') ?? pkg.getAttribute('lang');
    // try dc:language
    for (final e in opfDoc.findAllElements('language')) { lang = e.innerText; break; }
    return EpubMeta(title: title, author: author, language: lang);
  }
}
