package com.example.notesjava.common.security;

import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpStatus;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.oauth2.core.DelegatingOAuth2TokenValidator;
import org.springframework.security.oauth2.core.OAuth2Error;
import org.springframework.security.oauth2.core.OAuth2TokenValidator;
import org.springframework.security.oauth2.core.OAuth2TokenValidatorResult;
import org.springframework.security.oauth2.jose.jws.SignatureAlgorithm;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtClaimNames;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtIssuerValidator;
import org.springframework.security.oauth2.jwt.JwtTimestampValidator;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.util.StringUtils;

import java.util.ArrayList;
import java.util.List;

@Configuration
@EnableWebSecurity
@EnableConfigurationProperties(AuthProperties.class)
public class SecurityConfig {

    @Bean
    SecurityFilterChain apiSecurity(HttpSecurity http, ProblemDetailAuthenticationHandler handler) throws Exception {
        return http
                .csrf(AbstractHttpConfigurer::disable)
                .cors(Customizer.withDefaults())
                .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                .authorizeHttpRequests(requests -> requests
                        .requestMatchers("/actuator/health", "/actuator/health/**", "/actuator/info").permitAll()
                        .anyRequest().authenticated())
                .oauth2ResourceServer(server -> server
                        .jwt(Customizer.withDefaults())
                        .authenticationEntryPoint(handler)
                        .accessDeniedHandler(handler))
                .exceptionHandling(exceptions -> exceptions
                        .authenticationEntryPoint(handler)
                        .accessDeniedHandler(handler))
                .build();
    }

    @Bean
    JwtDecoder jwtDecoder(AuthProperties properties) {
        NimbusJwtDecoder decoder = NimbusJwtDecoder.withJwkSetUri(properties.jwkSetUri())
                .jwsAlgorithm(SignatureAlgorithm.ES256)
                .build();
        decoder.setJwtValidator(validator(properties));
        return decoder;
    }

    private static OAuth2TokenValidator<Jwt> validator(AuthProperties properties) {
        List<OAuth2TokenValidator<Jwt>> validators = new ArrayList<>();
        validators.add(new JwtTimestampValidator());
        validators.add(requiredClaim(JwtClaimNames.EXP));
        validators.add(requiredClaim(JwtClaimNames.SUB));
        if (StringUtils.hasText(properties.issuer())) {
            validators.add(new JwtIssuerValidator(properties.issuer()));
        }
        if (StringUtils.hasText(properties.audience())) {
            validators.add(audience(properties.audience()));
        }
        return new DelegatingOAuth2TokenValidator<>(validators);
    }

    private static OAuth2TokenValidator<Jwt> requiredClaim(String claim) {
        OAuth2Error error = new OAuth2Error("invalid_token", "The " + claim + " claim is required", null);
        return jwt -> jwt.hasClaim(claim)
                ? OAuth2TokenValidatorResult.success()
                : OAuth2TokenValidatorResult.failure(error);
    }

    private static OAuth2TokenValidator<Jwt> audience(String expected) {
        OAuth2Error error = new OAuth2Error("invalid_token", "The required audience is missing", null);
        return jwt -> jwt.getAudience() != null && jwt.getAudience().contains(expected)
                ? OAuth2TokenValidatorResult.success()
                : OAuth2TokenValidatorResult.failure(error);
    }

    @Bean
    ProblemDetailAuthenticationHandler problemDetailAuthenticationHandler() {
        return new ProblemDetailAuthenticationHandler(HttpStatus.UNAUTHORIZED, HttpStatus.FORBIDDEN);
    }
}
