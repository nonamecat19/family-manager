package com.example.notesjava.common.security;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.jspecify.annotations.NonNull;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ProblemDetail;
import org.springframework.security.access.AccessDeniedException;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.AuthenticationEntryPoint;
import org.springframework.security.web.access.AccessDeniedHandler;

import java.io.IOException;

import tools.jackson.databind.ObjectMapper;

public class ProblemDetailAuthenticationHandler implements AuthenticationEntryPoint, AccessDeniedHandler {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    private final HttpStatus unauthenticated;
    private final HttpStatus unauthorized;

    public ProblemDetailAuthenticationHandler(HttpStatus unauthenticated, HttpStatus unauthorized) {
        this.unauthenticated = unauthenticated;
        this.unauthorized = unauthorized;
    }

    @Override
    public void commence(@NonNull HttpServletRequest request, HttpServletResponse response,
                         @NonNull AuthenticationException authException) throws IOException {
        response.setHeader("WWW-Authenticate", "Bearer");
        write(response, unauthenticated, "Unauthenticated", "A valid bearer access token is required.");
    }

    @Override
    public void handle(@NonNull HttpServletRequest request, @NonNull HttpServletResponse response,
                       @NonNull AccessDeniedException accessDeniedException) throws IOException {
        write(response, unauthorized, "Forbidden", "The token does not grant access to this resource.");
    }

    private static void write(HttpServletResponse response, HttpStatus status, String title, String detail)
            throws IOException {
        ProblemDetail problem = ProblemDetail.forStatusAndDetail(status, detail);
        problem.setTitle(title);
        response.setStatus(status.value());
        response.setContentType(MediaType.APPLICATION_PROBLEM_JSON_VALUE);
        response.getWriter().write(MAPPER.writeValueAsString(problem));
    }
}
