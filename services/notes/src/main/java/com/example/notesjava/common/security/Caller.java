package com.example.notesjava.common.security;

import java.util.UUID;

public record Caller(String userId, UUID familyId, String email) {
}
