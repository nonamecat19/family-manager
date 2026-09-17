package com.example.notesjava.common.security;

import com.example.notesjava.common.error.MissingFamilyException;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
public class CallerContext {

    public Caller require() {
        Jwt jwt = token();
        String userId = jwt.getSubject();
        if (userId == null || userId.isBlank()) {
            throw new MissingFamilyException("The access token carries no subject.");
        }
        return new Caller(userId, familyId(jwt), jwt.getClaimAsString("email"));
    }

    public UUID requireFamilyId() {
        return require().familyId();
    }

    private static Jwt token() {
        Authentication authentication = SecurityContextHolder.getContext().getAuthentication();
        if (!(authentication instanceof JwtAuthenticationToken jwtAuthentication)) {
            throw new MissingFamilyException("The request carries no verified access token.");
        }
        return jwtAuthentication.getToken();
    }

    private static UUID familyId(Jwt jwt) {
        String claim = jwt.getClaimAsString("family_id");
        if (claim == null || claim.isBlank()) {
            throw new MissingFamilyException("The caller belongs to no family yet.");
        }
        try {
            return UUID.fromString(claim);
        } catch (IllegalArgumentException ex) {
            throw new MissingFamilyException("The family_id claim is not a UUID.");
        }
    }
}
