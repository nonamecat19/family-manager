package com.example.notesjava.common.security;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "notes.auth")
public record AuthProperties(String jwkSetUri, String issuer, String audience) {
}
