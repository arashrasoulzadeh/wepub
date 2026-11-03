import 'package:http/http.dart' as http;

class ApiClient {
  final String baseUrl;
  final String token;
  final http.Client client;
  ApiClient({required this.baseUrl, required this.token, http.Client? client})
      : client = client ?? http.Client();
  Future<http.Response> authCheck() =>
      client.get(Uri.parse('$baseUrl/v1/auth/check'), headers: {'Authorization': 'Bearer $token'});
}
