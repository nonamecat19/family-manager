package com.example.notesjava.common.connect;

import com.example.notesjava.support.AbstractIntegrationTest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.test.context.bean.override.mockito.MockitoBean;

import java.io.ByteArrayInputStream;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Instant;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;

@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class ConnectBodyLimitIT extends AbstractIntegrationTest {

    @LocalServerPort
    private int port;

    @MockitoBean
    private JwtDecoder jwtDecoder;

    @BeforeEach
    void token() {
        when(jwtDecoder.decode(anyString())).thenReturn(Jwt.withTokenValue("token")
                .header("alg", "ES256")
                .subject("11111111-1111-1111-1111-111111111111")
                .claim("family_id", "22222222-2222-2222-2222-222222222222")
                .issuedAt(Instant.now())
                .expiresAt(Instant.now().plusSeconds(60))
                .build());
    }

    private HttpResponse<String> chunked(byte[] body) throws Exception {
        HttpRequest request = HttpRequest.newBuilder(
                        URI.create("http://localhost:" + port + "/notes.v1.NotesService/ListNotebooks"))
                .version(HttpClient.Version.HTTP_1_1)
                .header("Authorization", "Bearer token")
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofInputStream(() -> new ByteArrayInputStream(body)))
                .build();
        return HttpClient.newHttpClient().send(request, HttpResponse.BodyHandlers.ofString());
    }

    @Test
    void aChunkedBodyOverTheCapIsCutOffAndRefused() throws Exception {
        HttpResponse<String> response = chunked(new byte[ConnectController.MAX_BODY_BYTES + (256 << 10)]);

        assertThat(response.statusCode()).isEqualTo(429);
        assertThat(response.body()).contains("\"code\":\"resource_exhausted\"");
    }

    @Test
    void aSmallChunkedBodyIsServed() throws Exception {
        HttpResponse<String> response = chunked("{}".getBytes());

        assertThat(response.statusCode()).isEqualTo(200);
        assertThat(response.body()).contains("notebooks");
    }
}
