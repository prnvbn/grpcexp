# testserver

dummy test server used for testing `grpcexp`

quick grpcurl command to test the service - `grpcurl -plaintext -d '{"message": "hello", "boolean": "true", "enum": "1"}' :50051 echo.v1.EchoService.Echo`
